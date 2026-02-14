package moods

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philopaterwaheed/phiocker/internal/download"
	"github.com/philopaterwaheed/phiocker/internal/utils"
)

func Download(basePath string) {
	if len(os.Args) < 3 {
		panic("usage: download <url>")
	}

	name := os.Args[2]
	imagePath := filepath.Join(basePath, "images", name, "rootfs")

	if _, err := os.Stat(imagePath); err == nil {
		if isEmpty, err := utils.IsDirectoryEmpty(imagePath); err == nil && !isEmpty {
			fmt.Printf("Image '%s' already exists, skipping download.\n", name)
			return
		}
	}

	fmt.Println("Downloading base image...")
	if err := download.PullAndExtractImage(name, imagePath); err != nil {
		panic(fmt.Sprintf("Failed to download/extract image: %v", err))
	}
}
