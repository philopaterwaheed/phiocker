package network

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	BridgeName = "phiocker0"
	BridgeIP   = "172.20.0.1"
	BridgeCIDR = "172.20.0.1/16"
	Subnet     = "172.20.0.0/16"
)

var (
	mu           sync.Mutex
	allocatedIPs = make(map[string]bool)
	bridgeReady  bool
)

func SetupBridge() error {
	if bridgeReady {
		return nil
	}

	if _, err := net.InterfaceByName(BridgeName); err != nil {
		if err := run("ip", "link", "add", "name", BridgeName, "type", "bridge"); err != nil {
			return fmt.Errorf("failed to create bridge: %v", err)
		}
	}

	iface, err := net.InterfaceByName(BridgeName)
	if err != nil {
		return fmt.Errorf("bridge interface not found after creation: %v", err)
	}
	addrs, _ := iface.Addrs()
	hasIP := false
	for _, a := range addrs {
		if strings.HasPrefix(a.String(), BridgeIP+"/") {
			hasIP = true
			break
		}
	}
	if !hasIP {
		if err := run("ip", "addr", "add", BridgeCIDR, "dev", BridgeName); err != nil {
			return fmt.Errorf("failed to assign IP to bridge: %v", err)
		}
	}
	if err := run("ip", "link", "set", BridgeName, "up"); err != nil {
		return fmt.Errorf("failed to bring bridge up: %v", err)
	}

	os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644)

	if runSilent("iptables", "-t", "nat", "-C", "POSTROUTING", "-s", Subnet, "!", "-o", BridgeName, "-j", "MASQUERADE") != nil {
		if err := run("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", Subnet, "!", "-o", BridgeName, "-j", "MASQUERADE"); err != nil {
			return fmt.Errorf("failed to set up NAT: %v", err)
		}
	}

	if runSilent("iptables", "-C", "FORWARD", "-i", BridgeName, "-j", "ACCEPT") != nil {
		run("iptables", "-I", "FORWARD", "1", "-i", BridgeName, "-j", "ACCEPT")
	}
	if runSilent("iptables", "-C", "FORWARD", "-o", BridgeName, "-j", "ACCEPT") != nil {
		run("iptables", "-I", "FORWARD", "1", "-o", BridgeName, "-j", "ACCEPT")
	}

	bridgeReady = true
	return nil
}

func allocateIPLocked() string {
	for third := 0; third < 256; third++ {
		start := 2
		if third > 0 {
			start = 1
		}
		for fourth := start; fourth < 256; fourth++ {
			ip := fmt.Sprintf("172.20.%d.%d", third, fourth)
			if !allocatedIPs[ip] {
				allocatedIPs[ip] = true
				return ip
			}
		}
	}
	return ""
}

func ReleaseIP(ip string) {
	mu.Lock()
	defer mu.Unlock()
	delete(allocatedIPs, ip)
}

type ContainerNetInfo struct {
	VethHost      string
	VethContainer string
	ContainerIP   string
}

// For unique veth names.
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func SetupContainerNetwork(pid int, rootfsPath string) (*ContainerNetInfo, error) {
	if err := SetupBridge(); err != nil {
		return nil, fmt.Errorf("bridge setup: %v", err)
	}

	mu.Lock()
	containerIP := allocateIPLocked()
	mu.Unlock()
	if containerIP == "" {
		return nil, fmt.Errorf("no free IPs available")
	}

	pidStr := strconv.Itoa(pid)
	suffix := randomHex(4)
	vethHost := "veth" + suffix
	vethPeer := "vpeer" + suffix

	// Create veth pair (both in host namespace initially)
	if err := run("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethPeer); err != nil {
		ReleaseIP(containerIP)
		return nil, fmt.Errorf("failed to create veth pair: %v", err)
	}

	// Attach host-side veth to bridge
	if err := run("ip", "link", "set", vethHost, "master", BridgeName); err != nil {
		cleanupVeth(vethHost, containerIP)
		return nil, fmt.Errorf("failed to attach veth to bridge: %v", err)
	}

	// Bring host-side veth up
	if err := run("ip", "link", "set", vethHost, "up"); err != nil {
		cleanupVeth(vethHost, containerIP)
		return nil, fmt.Errorf("failed to bring veth up: %v", err)
	}

	// Move peer veth into the container's network namespace
	if err := run("ip", "link", "set", vethPeer, "netns", pidStr); err != nil {
		cleanupVeth(vethHost, containerIP)
		return nil, fmt.Errorf("failed to move veth to container ns: %v", err)
	}

	// Configure networking inside the container's network namespace using nsenter
	nsenter := func(args ...string) error {
		fullArgs := append([]string{"-t", pidStr, "-n", "--"}, args...)
		return run("nsenter", fullArgs...)
	}

	// Rename the peer to eth0 inside the container namespace
	if err := nsenter("ip", "link", "set", vethPeer, "name", "eth0"); err != nil {
		cleanupVeth(vethHost, containerIP)
		return nil, fmt.Errorf("failed to rename veth to eth0: %v", err)
	}

	// Assign IP to container's eth0
	if err := nsenter("ip", "addr", "add", containerIP+"/16", "dev", "eth0"); err != nil {
		cleanupVeth(vethHost, containerIP)
		return nil, fmt.Errorf("failed to assign IP in container: %v", err)
	}

	// Bring up loopback
	if err := nsenter("ip", "link", "set", "lo", "up"); err != nil {
		cleanupVeth(vethHost, containerIP)
		return nil, fmt.Errorf("failed to bring up loopback: %v", err)
	}

	// Bring up eth0
	if err := nsenter("ip", "link", "set", "eth0", "up"); err != nil {
		cleanupVeth(vethHost, containerIP)
		return nil, fmt.Errorf("failed to bring up eth0: %v", err)
	}

	// Set it as the default gateway
	if err := nsenter("ip", "route", "add", "default", "via", BridgeIP); err != nil {
		cleanupVeth(vethHost, containerIP)
		return nil, fmt.Errorf("failed to set default route: %v", err)
	}

	setupDNS(rootfsPath)

	fmt.Printf("Container networking configured: IP=%s veth=%s\n", containerIP, vethHost)

	return &ContainerNetInfo{
		VethHost:      vethHost,
		VethContainer: "eth0",
		ContainerIP:   containerIP,
	}, nil
}

// TeardownContainerNetwork cleans up networking
func TeardownContainerNetwork(info *ContainerNetInfo) {
	if info == nil {
		return
	}

	run("ip", "link", "del", info.VethHost)

	ReleaseIP(info.ContainerIP)
}

func setupDNS(rootfsPath string) {
	etcPath := filepath.Join(rootfsPath, "etc")
	os.MkdirAll(etcPath, 0755)

	// falling back to public DNS
	nameservers := getHostNameservers()
	if len(nameservers) == 0 {
		nameservers = []string{"8.8.8.8", "8.8.4.4"}
	}

	var content strings.Builder
	content.WriteString("# Generated by phiocker\n")
	for _, ns := range nameservers {
		content.WriteString("nameserver " + ns + "\n")
	}

	resolvPath := filepath.Join(etcPath, "resolv.conf")
	os.WriteFile(resolvPath, []byte(content.String()), 0644)
}

func getHostNameservers() []string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}

	var nameservers []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Skip loopback nameservers (like 127.0.0.53 from systemd-resolved)
				// as they won't be reachable from the container's network namespace
				if !strings.HasPrefix(parts[1], "127.") {
					nameservers = append(nameservers, parts[1])
				}
			}
		}
	}
	return nameservers
}

func cleanupVeth(vethHost, ip string) {
	run("ip", "link", "del", vethHost)
	ReleaseIP(ip)
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %v: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
