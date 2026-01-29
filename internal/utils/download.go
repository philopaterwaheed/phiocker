package utils

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

type progressReader struct {
	reader io.Reader
	read   int64
	total  int64
	done   int32
}

// Read implements the io.Reader interface for progressReader.
func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	atomic.AddInt64(&p.read, int64(n))
	return n, err
}

func (p *progressReader) printProgress() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		if atomic.LoadInt32(&p.done) == 1 {
			return
		}
		read := atomic.LoadInt64(&p.read)
		percent := (float64(read) / float64(p.total)) * 100
		fmt.Printf(
			"\rDownloading: %.1f MB / %.1f MB (%.0f%%)",
			float64(read)/1024/1024,
			float64(p.total)/1024/1024,
			percent,
		)
	}
}

func DownloadAndExtract(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %s", resp.Status)
	}

	pr := &progressReader{
		reader: resp.Body,
		total:  resp.ContentLength,
	}
	go pr.printProgress()

	var extractErr error
	// every body reads from below
	// It calls pr.Read
	switch detectArchiveType(url) {
	case "tar.gz":
		extractErr = extractTarGz(pr, dest)
	case "tar.xz":
		extractErr = extractTarXz(pr, dest)
	case "zip":
		extractErr = extractZip(pr, dest)
	case "tar.zst":
		extractErr = extractTarZst(pr, dest)
	default:
		extractErr = fmt.Errorf("unsupported archive type")
	}

	atomic.StoreInt32(&pr.done, 1) // stop progress printing
	fmt.Println()
	return extractErr
}

func detectArchiveType(url string) string {
	ext := strings.ToLower(filepath.Ext(url))

	switch ext {
	case ".gz":
		return "tar.gz"
	case ".xz":
		return "tar.xz"
	case ".zip":
		return "zip"
	case ".zst":
		return "tar.zst"
	}
	return ""
}

func extractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	return extractTar(tar.NewReader(gz), dest)
}

func extractTarXz(r io.Reader, dest string) error {
	xzr, err := xz.NewReader(r)
	if err != nil {
		return err
	}
	return extractTar(tar.NewReader(xzr), dest)
}

func extractTarZst(r io.Reader, dest string) error {
	decoder, err := zstd.NewReader(r)
	if err != nil {
		return err
	}
	defer decoder.Close()

	tr := tar.NewReader(decoder)
	return extractTar(tr, dest)
}

func extractTar(tr *tar.Reader, dest string) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func extractZip(r io.Reader, dest string) error {
	buf, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return err
	}

	for _, f := range zr.File {
		target := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, f.Mode())
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
