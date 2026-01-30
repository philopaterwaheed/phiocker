package cmd

import (
	"os"
	"os/exec"
	"github.com/philopaterwaheed/phiocker/internal/errors")

func RunCmd(name string, args ...string) {
    cmd := exec.Command(name, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        errors.Must(err)
    }
}


