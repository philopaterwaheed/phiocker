package moods

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

const (
	cgroupRoot = "/sys/fs/cgroup"
	cgroupName = "phiocker"
)

type ContainerProcess struct {
	Cmd       *exec.Cmd
	CgPath    string
	StdinPipe io.WriteCloser
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
	if cp.StdinPipe != nil {
		cp.StdinPipe.Close()
	}
	if err := cp.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return cp.Cmd.Process.Kill()
	}
	return nil
}

func RunDetached(args []string) (*ContainerProcess, error) {
	cmd := exec.Command(
		"/proc/self/exe",
		append([]string{"child"}, args...)...,
	)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start container: %v", err)
	}

	cgPath := createCgroup(cmd.Process.Pid)

	return &ContainerProcess{
		Cmd:       cmd,
		CgPath:    cgPath,
		StdinPipe: stdinPipe,
	}, nil
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) {
	cmd := exec.Command(
		"/proc/self/exe",
		append([]string{"child"}, args...)...,
	)

	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

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

func createCgroup(pid int) string {
	cgPath := filepath.Join(cgroupRoot, cgroupName)
	fmt.Print(cgPath)

	if err := os.MkdirAll(cgPath, 0755); err != nil && !os.IsExist(err) {
		fmt.Println("err at MkdirAll:", err)
		panic(err)
	}
	writeFile(
		filepath.Join(cgroupRoot, "cgroup.subtree_control"),
		"+cpu +memory +pids",
	)
	writeFile(
		filepath.Join(cgPath, "cpu.max"),
		"50000 100000",
	)
	writeFile(
		filepath.Join(cgPath, "memory.max"),
		strconv.Itoa(100*1024*1024),
	)
	writeFile(
		filepath.Join(cgPath, "pids.max"),
		"20",
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
