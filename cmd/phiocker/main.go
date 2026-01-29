package main

import (
	"fmt"
	"github.com/philopaterwaheed/phiocker/internal/consts"
	"github.com/philopaterwaheed/phiocker/internal/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

const (
	cgroupRoot = "/sys/fs/cgroup"
	cgroupName = "phiocker"
	basePath   = "/var/lib/phiocker"
)

func main() {
	if len(os.Args) < 2 {
		panic("usage: run <command>")
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	case "download":
		download()
	default:
		panic("unknown command")
	}
}

func run() {
	cmd := exec.Command(
		"/proc/self/exe",
		append([]string{"child"}, os.Args[2:]...)...,
	)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	cgPath := createCgroup(cmd.Process.Pid)
	defer deleteCgroup(cgPath)

	if err := cmd.Wait(); err != nil {
		panic(err)
	}
}

func child() {
	fmt.Printf("Container started with PID %d\n", os.Getpid())

	if err := syscall.Chroot("/tmp/"); err != nil {
		fmt.Printf("err at Chroot: %v\n", err)
		panic(err)
	}
	if err := os.Chdir("/"); err != nil {
		fmt.Printf("err at chdir: %v\n", err)
		panic(err)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fmt.Printf("err at Mount: %v\n", err)
		panic(err)
	}

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(err)
	}

	syscall.Unmount("/proc", 0)
}
func download() {
	fmt.Println("Downloading base image...")
	if len(os.Args) < 3 {
		panic("usage: download <url>")
	}

	name := os.Args[2]
	var url string
	switch name {
	case "alpine":
		url = consts.Alpine_url
	case "ubuntu":
		url = consts.Ubuntu_url
	case "arch":
		url= consts.Arch_url
	default:
		panic("unknown image name")
	}
	fmt.Printf("Downloading %s from %s\n", name, url)
	if err := utils.DownloadAndExtract(url, filepath.Join(basePath, "images", name, "rootfs")); err != nil {
		must(err)
	}

}

func createCgroup(pid int) string {
	cgPath := filepath.Join(cgroupRoot, cgroupName)
	fmt.Print(cgPath)

	if err := os.MkdirAll(cgPath, 0755); err != nil && !os.IsExist(err) {
		fmt.Println("err at MkdirAll:", err)
		panic(err)
	}
	writeFile(
		filepath.Join(cgroupRoot, "cgroup.subtree_control"),
		"+cpu +memory",
	)
	writeFile(
		filepath.Join(cgPath, "cpu.max"),
		"50000 100000",
	)
	writeFile(
		filepath.Join(cgPath, "memory.max"),
		strconv.Itoa(100*1024*1024),
	)

	// Add process to cgroup
	writeFile(
		filepath.Join(cgPath, "cgroup.procs"),
		strconv.Itoa(pid),
	)
	fmt.Print("finished cgroup setup\n")

	return cgPath
}

func deleteCgroup(path string) {
	if err := os.Remove(path); err != nil {
		fmt.Println("warning: failed to remove cgroup:", err)
	}
}

func writeFile(path, value string) {
	if err := os.WriteFile(path, []byte(value), 0644); err != nil {
		panic(err)
	}
}

func must(err error) {
panic(err)
}
