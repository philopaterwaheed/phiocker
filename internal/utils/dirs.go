package utils

import (
	"os"
	"path/filepath"
)

func IsDirectoryEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func CopyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		} else if info.Mode()&os.ModeSymlink != 0 {
			// Handle symlinks
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			os.Remove(dstPath)
			return os.Symlink(linkTarget, dstPath)
		} else {
			return CopyFile(path, dstPath)
		}
	})
}

func CalculateDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}