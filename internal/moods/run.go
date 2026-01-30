package moods

import (
	"fmt"
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

func Run() {
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
