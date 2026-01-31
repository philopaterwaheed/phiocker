package moods

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"path/filepath"
)

func Child(name, basePath string) {
	fmt.Printf("Container started with PID %d\n", os.Getpid())
	path := filepath.Join(basePath, "containers", name, "rootfs")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(name + " container does not exist")
	}

	if err := syscall.Chroot(path); err != nil {
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

	cmd := exec.Command("/bin/sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(err)
	}

	syscall.Unmount("/proc", 0)
}

