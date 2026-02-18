package moods

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/creack/pty"
	"github.com/philopaterwaheed/phiocker/internal/utils"
)

const (
	cgroupRoot = "/sys/fs/cgroup"
	cgroupName = "phiocker"
)

type ContainerProcess struct {
	Cmd       *exec.Cmd
	CgPath    string
	StdinPipe io.WriteCloser
	PTYMaster *os.File // PTY master fd
}

func (cp *ContainerProcess) PID() int {
	return cp.Cmd.Process.Pid
}

func (cp *ContainerProcess) Wait() error {
	err := cp.Cmd.Wait()
	deleteCgroup(cp.CgPath)
	return err
}

func (cp *ContainerProcess) Stop() error {
	if cp.PTYMaster != nil {
		cp.PTYMaster.Close()
	}
	if cp.StdinPipe != nil {
		cp.StdinPipe.Close()
	}
	if err := cp.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return cp.Cmd.Process.Kill()
	}
	return nil
}

func RunDetached(args []string, basePath string) (*ContainerProcess, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("missing container name")
	}
	containerName := args[0]

	configPath := filepath.Join(basePath, "containers", containerName, "config.json")
	var limits Limits
	if configFile, err := utils.OpenFile(configPath); err == nil {
		config := LoadConfig(configFile)
		limits = config.Limits
		configFile.Close()
	}

	cgPath, cgFile := setupCgroup(limits)
	defer cgFile.Close()

	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to create PTY: %v", err)
	}

	cmd := exec.Command(
		"/proc/self/exe",
		append([]string{"child"}, args...)...,
	)

	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0, // child fd 0 (stdin) = PTY slave
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
		UseCgroupFD: true,
		CgroupFD:    int(cgFile.Fd()),
	}

	if err := cmd.Start(); err != nil {
		ptmx.Close()
		tty.Close()
		return nil, fmt.Errorf("failed to start container: %v", err)
	}

	// Close slave in parent — child has its own copy
	tty.Close()

	// Set a sensible default terminal size
	utils.SetPTYWinSize(ptmx, 24, 80)

	return &ContainerProcess{
		Cmd:       cmd,
		CgPath:    cgPath,
		PTYMaster: ptmx,
	}, nil
}

func setupCgroup(limits Limits) (string, *os.File) {
	cgPath := filepath.Join(cgroupRoot, cgroupName)

	if err := os.MkdirAll(cgPath, 0755); err != nil && !os.IsExist(err) {
		fmt.Println("err at MkdirAll:", err)
		panic(err)
	}
	writeFile(
		filepath.Join(cgroupRoot, "cgroup.subtree_control"),
		"+cpu +memory +pids",
	)

	cpuQuota := 50000
	cpuPeriod := 100000
	if limits.CPUQuota > 0 {
		cpuQuota = limits.CPUQuota
	}
	if limits.CPUPeriod > 0 {
		cpuPeriod = limits.CPUPeriod
	}
	writeFile(
		filepath.Join(cgPath, "cpu.max"),
		fmt.Sprintf("%d %d", cpuQuota, cpuPeriod),
	)

	memoryLimit := 100 * 1024 * 1024
	if limits.Memory > 0 {
		memoryLimit = limits.Memory
	}
	writeFile(
		filepath.Join(cgPath, "memory.max"),
		strconv.Itoa(memoryLimit),
	)

	pidLimit := 20
	if limits.PIDs > 0 {
		pidLimit = limits.PIDs
	}
	writeFile(
		filepath.Join(cgPath, "pids.max"),
		strconv.Itoa(pidLimit),
	)

	cgFile, err := os.Open(cgPath)
	if err != nil {
		panic(fmt.Sprintf("failed to open cgroup dir: %v", err))
	}
	fmt.Print("finished cgroup setup\n")

	return cgPath, cgFile
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
