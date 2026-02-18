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

	// Check if base image exists, if not download it
	imagePath := filepath.Join(basePath, "images", baseimage, "rootfs")
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		fmt.Printf("Base image '%s' not found, downloading...\n", baseimage)
		errors.Must(os.MkdirAll(filepath.Dir(imagePath), 0755))
		errors.Must(download.PullAndExtractImage(baseimage, imagePath))
		fmt.Printf("Base image '%s' downloaded successfully.\n", baseimage)
	} else if err != nil {
		panic(fmt.Sprintf("Error checking base image: %v", err))
	} else {
		if isEmpty, err := utils.IsDirectoryEmpty(imagePath); err == nil && isEmpty {
			fmt.Printf("Base image '%s' directory is empty, re-downloading...\n", baseimage)
			if err := download.PullAndExtractImage(baseimage, imagePath); err != nil {
				panic(fmt.Sprintf("Failed to download base image: %v", err))
			}
			fmt.Printf("Base image '%s' downloaded successfully.\n", baseimage)
		} else {
			fmt.Printf("Using existing base image '%s'.\n", baseimage)
		}
	}

	errors.Must(os.MkdirAll(containerPath, 0755))

	if err := utils.CopyDirectory(imagePath, containerPath); err != nil {
		panic(fmt.Sprintf("Failed to copy image to container: %v", err))
	}

	if len(config.Copy) > 0 {
		fmt.Printf("Copying %d file(s) to container...\n", len(config.Copy))
		for _, copySpec := range config.Copy {
			srcPath := copySpec.Src
			if !filepath.IsAbs(srcPath) {
				configDir := filepath.Dir(file.Path)
				srcPath = filepath.Join(configDir, srcPath)
			}

			dstPath := filepath.Join(containerPath, copySpec.Dst)

			info, err := os.Lstat(srcPath)
			if err != nil {
				panic(fmt.Sprintf("Failed to stat source '%s': %v", srcPath, err))
			}

			if info.IsDir() {
				fmt.Printf("  Copying directory %s -> %s\n", srcPath, copySpec.Dst)
				if err := utils.CopyDirectory(srcPath, dstPath); err != nil {
					panic(fmt.Sprintf("Failed to copy directory '%s' to '%s': %v", srcPath, copySpec.Dst, err))
				}
			} else {
				fmt.Printf("  Copying %s -> %s\n", srcPath, copySpec.Dst)
				if err := utils.CopyFile(srcPath, dstPath); err != nil {
					panic(fmt.Sprintf("Failed to copy '%s' to '%s': %v", srcPath, copySpec.Dst, err))
				}
			}
		}
		fmt.Println("File copying completed.")
	}

	if config.Workdir != "" {
		workdirPath := filepath.Join(containerPath, config.Workdir)
		if err := os.MkdirAll(workdirPath, 0755); err != nil {
			fmt.Printf("Warning: Failed to create workdir '%s': %v\n", config.Workdir, err)
		} else {
			fmt.Printf("Created working directory: %s\n", config.Workdir)
		}
	}

	cmd.RunCmd("cp", file.Path, filepath.Join(basePath, "containers", name, "config.json"))
	fmt.Printf("Container %s created successfully!\n", name)
}
