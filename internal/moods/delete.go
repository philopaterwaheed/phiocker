package moods

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philopaterwaheed/phiocker/internal/utils"
)

func DeleteContainer(containerName, basePath string) error {
	containerPath := filepath.Join(basePath, "containers", containerName)

	if _, err := os.Stat(containerPath); os.IsNotExist(err) {
		fmt.Printf("Container '%s' does not exist.\n", containerName)
		return nil
	} else if err != nil {
		
		return fmt.Errorf("operation failed: %v", err)
	}

	info, err := os.Stat(containerPath)
	if err != nil {
		return fmt.Errorf("operation failed: %v", err)
	}
	if !info.IsDir() {
		fmt.Printf("'%s' is not a valid container directory.\n", containerName)
		return nil
	}

	fmt.Printf("Container '%s' found at: %s\n", containerName, containerPath)

	size, err := utils.CalculateDirectorySize(containerPath)
	if err == nil && size > 100*1024*1024 {
		fmt.Printf("Warning: Container is large (%.2f MB)\n", float64(size)/(1024*1024))
	}

	fmt.Printf("Deleting container '%s'...\n", containerName)
	if err := os.RemoveAll(containerPath); err != nil {
		return fmt.Errorf("failed to delete container '%s': %v", containerName, err)
	}

	fmt.Printf("Container '%s' has been successfully deleted.\n", containerName)
return nil
}

func DeleteAllContainers(basePath string) error {
	containersPath := filepath.Join(basePath, "containers")

	if _, err := os.Stat(containersPath); os.IsNotExist(err) {
		fmt.Println("No containers directory found.")
		return nil
	} else if err != nil {
		return fmt.Errorf("operation failed: %v", err)
	}

	entries, err := os.ReadDir(containersPath)
	if err != nil {
		return fmt.Errorf("operation failed: %v", err)
	}

	if len(entries) == 0 {
		fmt.Println("No containers found to delete.")
		return nil
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
	return nil
}

func DeleteImage(imageName, basePath string) error {
	imagePath := filepath.Join(basePath, "images", imageName)

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		fmt.Printf("Image '%s' does not exist.\n", imageName)
		return nil
	} else if err != nil {
		return fmt.Errorf("operation failed: %v", err)
	}

	info, err := os.Stat(imagePath)
	if err != nil {
		return fmt.Errorf("operation failed: %v", err)
	}
	if !info.IsDir() {
		fmt.Printf("'%s' is not a valid image directory.\n", imageName)
		return nil
	}

	fmt.Printf("Image '%s' found at: %s\n", imageName, imagePath)

	size, err := utils.CalculateDirectorySize(imagePath)
	if err == nil && size > 100*1024*1024 {
		fmt.Printf("Warning: Image is large (%.2f MB)\n", float64(size)/(1024*1024))
	}

	fmt.Printf("Deleting image '%s'...\n", imageName)
	if err := os.RemoveAll(imagePath); err != nil {
		return fmt.Errorf("failed to delete image '%s': %v", imageName, err)
	}

	fmt.Printf("Image '%s' has been successfully deleted.\n", imageName)
	return nil
}

func DeleteAllImages(basePath string) error {
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
		fmt.Println("No images found to delete.")
		return nil
	}

	fmt.Printf("Found %d image(s) to delete:\n", len(entries))
	totalSize := int64(0)
	imageNames := []string{}

	for _, entry := range entries {
		if entry.IsDir() {
			imageNames = append(imageNames, entry.Name())
			imagePath := filepath.Join(imagesPath, entry.Name())
			size, err := utils.CalculateDirectorySize(imagePath)
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

	successCount := 0
	for _, name := range imageNames {
		imagePath := filepath.Join(imagesPath, name)
		fmt.Printf("Deleting image '%s'...\n", name)
		if err := os.RemoveAll(imagePath); err != nil {
			fmt.Printf("Failed to delete image '%s': %v\n", name, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("Successfully deleted %d out of %d images.\n", successCount, len(imageNames))
	return nil
}
