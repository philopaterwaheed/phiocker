package moods

import (
	"fmt"
	"github.com/philopaterwaheed/phiocker/internal/download"
	"os"
	"path/filepath"
)

func Download(basePath string) {
	fmt.Println("Downloading base image...")
	if len(os.Args) < 3 {
		panic("usage: download <url>")
	}

	name := os.Args[2]
		if err := download.PullAndExtractImage(name, filepath.Join(basePath, "images", name, "rootfs")); err != nil {
			panic(fmt.Sprintf("Failed to download/extract image: %v", err))
		}

}
