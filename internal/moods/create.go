package moods

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philopaterwaheed/phiocker/internal/cmd"
	"github.com/philopaterwaheed/phiocker/internal/download"
	"github.com/philopaterwaheed/phiocker/internal/errors"
	"github.com/philopaterwaheed/phiocker/internal/utils"
)

func Create(generatorFilePath, basePath string) {
	file, err := utils.OpenFile(generatorFilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	config := LoadConfig(file)
	name := config.Name
	baseimage := config.Baseimage

	fmt.Printf("Creating container %s from image %s...\n", name, baseimage)

	containerPath := filepath.Join(basePath, "containers", name, "rootfs")
	if _, err := os.Stat(containerPath); err == nil {
		panic(fmt.Sprintf("Container '%s' already exists.", name))
	}
	if err := os.MkdirAll(containerPath, 0755); err != nil {
		errors.Must(err)
	}
	fmt.Printf("Pulling and extracting Docker image %s...\n", baseimage)
	if err := download.PullAndExtractImage(baseimage, containerPath); err != nil {
		panic(fmt.Sprintf("Failed to pull/extract image: %v", err))
	}
	cmd.RunCmd("cp", file.Path, filepath.Join(basePath, "containers", name, "config.json"))
	fmt.Printf("Container %s created successfully!\n", name)
}
