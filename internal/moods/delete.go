package moods

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philopaterwaheed/phiocker/internal/errors"
	"github.com/philopaterwaheed/phiocker/internal/utils"
)

func DeleteContainer(containerName, basePath string) {
	containerPath := filepath.Join(basePath, "containers", containerName)

	if _, err := os.Stat(containerPath); os.IsNotExist(err) {
		fmt.Printf("Container '%s' does not exist.\n", containerName)
		return
	} else if err != nil {
		errors.Must(err)
	}

	info, err := os.Stat(containerPath)
	if err != nil {
		errors.Must(err)
	}
	if !info.IsDir() {
		fmt.Printf("'%s' is not a valid container directory.\n", containerName)
		return
	}

	fmt.Printf("Container '%s' found at: %s\n", containerName, containerPath)

	size, err := utils.CalculateDirectorySize(containerPath)
	if err == nil && size > 100*1024*1024 {
		fmt.Printf("Warning: Container is large (%.2f MB)\n", float64(size)/(1024*1024))
	}

	if !utils.PromptForConfirmation(fmt.Sprintf("Are you sure you want to delete container '%s'?", containerName)) {
		fmt.Println("Deletion cancelled.")
		return
	}

	fmt.Printf("Deleting container '%s'...\n", containerName)
	if err := os.RemoveAll(containerPath); err != nil {
		errors.Must(fmt.Errorf("failed to delete container '%s': %v", containerName, err))
	}

	fmt.Printf("Container '%s' has been successfully deleted.\n", containerName)
}

func DeleteAllContainers(basePath string) {
	containersPath := filepath.Join(basePath, "containers")

	if _, err := os.Stat(containersPath); os.IsNotExist(err) {
		fmt.Println("No containers directory found.")
		return
	} else if err != nil {
		errors.Must(err)
	}

	entries, err := os.ReadDir(containersPath)
	if err != nil {
		errors.Must(err)
	}

	if len(entries) == 0 {
		fmt.Println("No containers found to delete.")
		return
	}

	fmt.Printf("Found %d container(s) to delete:\n", len(entries))
	totalSize := int64(0)
	containerNames := []string{}

	for _, entry := range entries {
		if entry.IsDir() {
			containerNames = append(containerNames, entry.Name())
			containerPath := filepath.Join(containersPath, entry.Name())
			size, err := utils.CalculateDirectorySize(containerPath)
			sizeStr := "unknown size"
			if err == nil {
				totalSize += size
				if size < 1024*1024 {
					sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
				} else {
					sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
				}
			}
			fmt.Printf("  - %s (%s)\n", entry.Name(), sizeStr)
		}
	}

	if totalSize > 0 {
		fmt.Printf("Total size: %.2f MB\n", float64(totalSize)/(1024*1024))
	}

	if !utils.PromptForConfirmation("Are you ABSOLUTELY sure you want to delete ALL containers? This cannot be undone!") {
		fmt.Println("Deletion cancelled.")
		return
	}

	if !utils.PromptForConfirmation(fmt.Sprintf("Type 'delete all containers' to confirm deletion of %d containers", len(containerNames))) {
		fmt.Println("Deletion cancelled.")
		return
	}

	successCount := 0
	for _, name := range containerNames {
		containerPath := filepath.Join(containersPath, name)
		fmt.Printf("Deleting container '%s'...\n", name)
		if err := os.RemoveAll(containerPath); err != nil {
			fmt.Printf("Failed to delete container '%s': %v\n", name, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("Successfully deleted %d out of %d containers.\n", successCount, len(containerNames))
}


