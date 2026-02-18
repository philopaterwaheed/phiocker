package moods

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philopaterwaheed/phiocker/internal/cmd"
	"github.com/philopaterwaheed/phiocker/internal/download"
	"github.com/philopaterwaheed/phiocker/internal/utils"
)

func Create(generatorFilePath, basePath string) error {
	file, err := utils.OpenFile(generatorFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	config := LoadConfig(file)
	name := config.Name
	baseimage := config.Baseimage

	fmt.Printf("Creating container %s from image %s...\n", name, baseimage)

	containerPath := filepath.Join(basePath, "containers", name, "rootfs")
	if _, err := os.Stat(containerPath); err == nil {
		return fmt.Errorf("container '%s' already exists", name)
	}

	// Check if base image exists, if not download it
	imagePath := filepath.Join(basePath, "images", baseimage, "rootfs")
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		fmt.Printf("Base image '%s' not found, downloading...\n", baseimage)
		if err := os.MkdirAll(filepath.Dir(imagePath), 0755); err != nil {
			return fmt.Errorf("failed to create image directory: %v", err)
		}
		if err := download.PullAndExtractImage(baseimage, imagePath); err != nil {
			return fmt.Errorf("failed to download base image: %v", err)
		}
		fmt.Printf("Base image '%s' downloaded successfully.\n", baseimage)
	} else if err != nil {
		return fmt.Errorf("error checking base image: %v", err)
	} else {
		if isEmpty, err := utils.IsDirectoryEmpty(imagePath); err == nil && isEmpty {
			fmt.Printf("Base image '%s' directory is empty, re-downloading...\n", baseimage)
			if err := download.PullAndExtractImage(baseimage, imagePath); err != nil {
				return fmt.Errorf("failed to download base image: %v", err)
			}
			fmt.Printf("Base image '%s' downloaded successfully.\n", baseimage)
		} else {
			fmt.Printf("Using existing base image '%s'.\n", baseimage)
		}
	}

	if err := os.MkdirAll(containerPath, 0755); err != nil {
		return fmt.Errorf("failed to create container directory: %v", err)
	}

	if err := utils.CopyDirectory(imagePath, containerPath); err != nil {
		return fmt.Errorf("failed to copy image to container: %v", err)
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
				return fmt.Errorf("failed to stat source '%s': %v", srcPath, err)
			}

			if info.IsDir() {
				fmt.Printf("  Copying directory %s -> %s\n", srcPath, copySpec.Dst)
				if err := utils.CopyDirectory(srcPath, dstPath); err != nil {
					return fmt.Errorf("failed to copy directory '%s' to '%s': %v", srcPath, copySpec.Dst, err)
				}
			} else {
				fmt.Printf("  Copying %s -> %s\n", srcPath, copySpec.Dst)
				if err := utils.CopyFile(srcPath, dstPath); err != nil {
					return fmt.Errorf("failed to copy '%s' to '%s': %v", srcPath, copySpec.Dst, err)
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
	return nil
}
