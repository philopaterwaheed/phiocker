package moods
import (
	"os"
	"syscall"
	"os/exec"
	"fmt"
)

func Child() {
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

