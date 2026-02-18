package moods

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philopaterwaheed/phiocker/internal/download"
	"github.com/philopaterwaheed/phiocker/internal/utils"
)

func UpdateImage(imageName, basePath string) error {
	imagePath := filepath.Join(basePath, "images", imageName, "rootfs")

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		fmt.Printf("Image '%s' does not exist. Use 'download' to download it first.\n", imageName)
		return nil
	} else if err != nil {
		return fmt.Errorf("operation failed: %v", err)
	}

	fmt.Printf("Image '%s' found.\n", imageName)

	size, err := utils.CalculateDirectorySize(imagePath)
	if err == nil {
		if size < 1024*1024 {
			fmt.Printf("Current size: %.1f KB\n", float64(size)/1024)
		} else {
			fmt.Printf("Current size: %.1f MB\n", float64(size)/(1024*1024))
		}
	}


	fmt.Printf("Removing old version of image '%s'...\n", imageName)
	if err := os.RemoveAll(imagePath); err != nil {
		return fmt.Errorf("failed to remove old image: %v", err)
	}

	fmt.Printf("Downloading updated image '%s'...\n", imageName)
	if err := download.PullAndExtractImage(imageName, imagePath); err != nil {
		return fmt.Errorf("failed to download/extract image: %v", err)
	}

	fmt.Printf("Image '%s' has been successfully updated.\n", imageName)
return nil
}

func UpdateAllImages(basePath string) error {
	imagesPath := filepath.Join(basePath, "images")

	if _, err := os.Stat(imagesPath); os.IsNotExist(err) {
		fmt.Println("No images directory found.")
		return nil
	} else if err != nil {
		return fmt.Errorf("operation failed: %v", err)
	}

	entries, err := os.ReadDir(imagesPath)
	if err != nil {
		return fmt.Errorf("operation failed: %v", err)
	}

	if len(entries) == 0 {
		fmt.Println("No images found to update.")
		return nil
	}

	imageNames := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			imageNames = append(imageNames, entry.Name())
		}
	}

	if len(imageNames) == 0 {
		fmt.Println("No images found to update.")
		return nil
	}

	fmt.Printf("Found %d image(s) to update:\n", len(imageNames))
	for _, name := range imageNames {
		imagePath := filepath.Join(imagesPath, name, "rootfs")
		size, err := utils.CalculateDirectorySize(imagePath)
		sizeStr := "unknown size"
		if err == nil {
			if size < 1024*1024 {
				sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
			} else {
				sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
			}
		}
		fmt.Printf("  - %s (%s)\n", name, sizeStr)
	}


	successCount := 0
	failCount := 0

	for _, name := range imageNames {
		fmt.Printf("\nUpdating image '%s'...\n", name)
		imagePath := filepath.Join(imagesPath, name, "rootfs")

		fmt.Printf("Removing old version...\n")
		if err := os.RemoveAll(imagePath); err != nil {
			fmt.Printf("Failed to remove old image '%s': %v\n", name, err)
			failCount++
			continue
		}

		fmt.Printf("Downloading updated version...\n")
		if err := download.PullAndExtractImage(name, imagePath); err != nil {
			fmt.Printf("Failed to download image '%s': %v\n", name, err)
			failCount++
			continue
		}

		fmt.Printf("Image '%s' updated successfully.\n", name)
		successCount++
	}

	fmt.Printf("\nUpdate complete: %d succeeded, %d failed.\n", successCount, failCount)
return nil
}
