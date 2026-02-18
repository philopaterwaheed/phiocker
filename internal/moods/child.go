package moods

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"github.com/philopaterwaheed/phiocker/internal/utils"
)

func Child(name, basePath string) {
	fmt.Printf("Container started with PID %d\n", os.Getpid())
	path := filepath.Join(basePath, "containers", name, "rootfs")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(name + " container does not exist")
	}
	
	configPath := filepath.Join(basePath, "containers", name, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic(name + " container config does not exist")
	}
	file, err := utils.OpenFile(configPath)
	if err != nil {
		panic(err)
	}
	config := LoadConfig(file)
	command := config.Cmd
	if err := syscall.Chroot(path); err != nil {
		fmt.Printf("err at Chroot: %v\n", err)
		panic(err)
	}
	
	workdir := "/"
	if config.Workdir != "" {
		workdir = config.Workdir
	}
	if err := os.Chdir(workdir); err != nil {
		fmt.Printf("err at chdir to %s: %v\n", workdir, err)
		panic(err)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fmt.Printf("err at Mount: %v\n", err)
		panic(err)
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(err)
	}

	syscall.Unmount("/proc", 0)
}

