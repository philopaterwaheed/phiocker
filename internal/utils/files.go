package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"io"
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

func CopyFile(src, dst string) error {
	sourceInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		os.Remove(dst)
		return os.Symlink(linkTarget, dst)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
