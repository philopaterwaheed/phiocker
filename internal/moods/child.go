package moods

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/philopaterwaheed/phiocker/internal/utils"
)

func bindMountDevice(rootfs, device string) error {
	target := filepath.Join(rootfs, device)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	file.Close()
	return syscall.Mount(device, target, "", syscall.MS_BIND, "")
}

func setupDevices(rootfs string) error {
	if err := os.MkdirAll(filepath.Join(rootfs, "dev"), 0755); err != nil {
		return err
	}
	for _, device := range []string{"/dev/null", "/dev/zero", "/dev/random", "/dev/urandom", "/dev/tty"} {
		if err := bindMountDevice(rootfs, device); err != nil {
			return fmt.Errorf("failed to bind mount %s: %v", device, err)
		}
	}
	return nil
}

func Child(name, basePath string) {
	fmt.Printf("Container started with PID %d\n", os.Getpid())
	path := filepath.Join(basePath, "containers", name, "rootfs")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(name + " container does not exist")
	}
	if err := repairContainerFilesystem(path); err != nil {
		panic(err)
	}
	if err := setupDevices(path); err != nil {
		panic(err)
	}

	configPath := filepath.Join(basePath, "containers", name, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic(name + " container config does not exist")
	}
	file, err := utils.OpenFile(configPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
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
	defer syscall.Unmount("/proc", 0)

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		panic(err)
	}
}
