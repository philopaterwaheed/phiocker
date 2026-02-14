package moods

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philopaterwaheed/phiocker/internal/errors"
	"github.com/philopaterwaheed/phiocker/internal/utils"
)

func ListContainers(basePath string) {
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
		fmt.Println("No containers found.")
		return
	}

	fmt.Println("Available containers:")
	for _, entry := range entries {
		if entry.IsDir() {
			containerPath := filepath.Join(containersPath, entry.Name())
			size, err := utils.CalculateDirectorySize(containerPath)
			sizeStr := "unknown size"
			if err == nil {
				if size < 1024*1024 {
					sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
				} else {
					sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
				}
			}
			fmt.Printf("  - %s (%s)\n", entry.Name(), sizeStr)
		}
	}
}
