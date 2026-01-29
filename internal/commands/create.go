package commands

import (
	"fmt"
	"github.com/philopaterwaheed/phiocker/internal/errors"
	"github.com/philopaterwaheed/phiocker/internal/download"
	"os"
	"os/exec"
	"path/filepath"
)

func Create(baseimage, name, basePath string) {
	fmt.Printf("Creating container %s from image %s...\n", name, baseimage)

	imagePath := filepath.Join(basePath, "images", baseimage, "rootfs")
	containerPath := filepath.Join(basePath, "containers", name, "rootfs")

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		download.DownloadAndExtract(baseimage, imagePath)
	}

	if _, err := os.Stat(containerPath); err == nil {
		panic(fmt.Sprintf("Container '%s' already exists.", name))
	}

	if err := os.MkdirAll(containerPath, 0755); err != nil {
		errors.Must(err)
	}

	// Copy base image rootfs to container rootfs
	cmd := exec.Command("cp", "-a", imagePath+"/.", containerPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errors.Must(err)
	}

	fmt.Printf("Container %s created successfully!\n", name)
}
