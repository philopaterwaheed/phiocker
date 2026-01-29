package commands

import (
	"fmt"
	"github.com/philopaterwaheed/phiocker/internal/download"
	"github.com/philopaterwaheed/phiocker/internal/errors"
	"os"
	"path/filepath"
)

func Download(basePath string) {
	fmt.Println("Downloading base image...")
	if len(os.Args) < 3 {
		panic("usage: download <url>")
	}

	name := os.Args[2]
		if url, ok := download.SupportedImages_links[name]; ok {
		fmt.Printf("Downloading %s from %s\n", name, url)
		if err := download.DownloadAndExtract(url, filepath.Join(basePath, "images", name, "rootfs")); err != nil {
			errors.Must(err)
	}
	} else {
		panic("unsupported image name")
	}

}
