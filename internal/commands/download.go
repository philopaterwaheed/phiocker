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
	var url string
	switch name {
	case "alpine":
		url = download.Alpine_url
	case "ubuntu":
		url = download.Ubuntu_url
	case "arch":
		url= download.Arch_url
	default:
		panic("unknown image name")
	}
	fmt.Printf("Downloading %s from %s\n", name, url)
	if err := download.DownloadAndExtract(url, filepath.Join(basePath, "images", name, "rootfs")); err != nil {
		errors.Must(err)
	}

}
