package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

const (
	url  = "https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/x86_64/alpine-minirootfs-latest-x86_64.tar.gz"
	dest = "/var/lib/phiocker/images/alpine/rootfs"
)

func downloadAndExtract(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %s", resp.Status)
	}

	total := resp.ContentLength
	if total <= 0 {
		return fmt.Errorf("server did not send Content-Length")
	}

	pr := &progressReader{
		reader: resp.Body,
		total:  total,
		start:  time.Now(),
	}

	go pr.printProgress()

	// It calls pr.Read
	gz, err := gzip.NewReader(pr)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	// every body reads from below

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

	pr.done = true
	return nil
}


type progressReader struct {
	reader io.Reader
	total  int64
	read   int64
	start  time.Time
	done   bool
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
		if p.done {
			return
		}

		read := atomic.LoadInt64(&p.read)
		percent := float64(read) / float64(p.total) * 100

		fmt.Printf(
			"\rDownloading: %.1f MB / %.1f MB (%.0f%%)",
			float64(read)/1024/1024,
			float64(p.total)/1024/1024,
			percent,
		)
	}
}
