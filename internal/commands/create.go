package commands

import (
	"encoding/json"
	"fmt"
	"github.com/philopaterwaheed/phiocker/internal/download"
	"github.com/philopaterwaheed/phiocker/internal/errors"
	"github.com/philopaterwaheed/phiocker/internal/cmd"
	"os"
	"path/filepath"
)

type ContainerConfig struct {
	Name      string  `json:"name"`
	Baseimage string  `json:"baseImage"`
}

func resolvePath(p string) (string, error) {
	if !filepath.IsAbs(p) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		p = filepath.Join(cwd, p)
	}

	p = filepath.Clean(p)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %s", p)
	}
	if filepath.Ext(p) != ".json" {
		panic("generator file must be a .json file")
	}

	return p, nil
}

func Create(generatorFilePath, basePath string) {
	absloteGeneratorFilePath, err := resolvePath(generatorFilePath)
	file, err := os.Open(absloteGeneratorFilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	var config ContainerConfig
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		panic(err)
	}
	name := config.Name
	baseimage := config.Baseimage

	fmt.Printf("Creating container %s from image %s...\n", name, baseimage)

	imagePath := filepath.Join(basePath, "images", baseimage, "rootfs")
	containerPath := filepath.Join(basePath, "containers", name, "rootfs")
	if baseimage == "arch" {
		imagePath = filepath.Join(imagePath, "root.x86_64")
	}

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		download.DownloadAndExtract(baseimage, imagePath)
	}

	if _, err := os.Stat(containerPath); err == nil {
		panic(fmt.Sprintf("Container '%s' already exists.", name))
	}

	if err := os.MkdirAll(containerPath, 0755); err != nil {
		errors.Must(err)
	}

	// Copy base image rootfs to container rootfs
	cmd.RunCmd("cp", "-a", imagePath+"/.", containerPath)
	cmd.RunCmd("cp", absloteGeneratorFilePath, filepath.Join(basePath, "containers", name, "config.json"))
	fmt.Printf("Container %s created successfully!\n", name)
}
