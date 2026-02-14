package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileWrapper struct {
	Path string
	*os.File
}
func ResolvePath(p string) (string, error) {
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

func OpenFile(path string) (*FileWrapper, error) {
	absolutePath, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(absolutePath)
	if err != nil {
		return nil, err
	}

	return &FileWrapper{
		Path: absolutePath,
		File: file,
	}, nil
}
