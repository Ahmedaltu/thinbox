// Package image extracts container root filesystems from tarball images.
package image

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const rootfsDir = "/var/lib/thinbox/rootfs"

// Extract unpacks the gzipped tar image at tarPath into a fresh directory
// under rootfsDir and returns its path. The caller owns cleanup
// (os.RemoveAll) once the container exits.
func Extract(tarPath string) (string, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return "", fmt.Errorf("image: open %s: %w", tarPath, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("image: gzip: %w", err)
	}
	defer gz.Close()

	if err := os.MkdirAll(rootfsDir, 0700); err != nil {
		return "", fmt.Errorf("image: mkdir %s: %w", rootfsDir, err)
	}
	dir, err := os.MkdirTemp(rootfsDir, "*")
	if err != nil {
		return "", fmt.Errorf("image: mkdtemp: %w", err)
	}

	if err := extractTar(tar.NewReader(gz), dir); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func extractTar(tr *tar.Reader, dir string) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("image: tar: %w", err)
		}

		// Reject entries that would escape dir (zip-slip).
		target := filepath.Join(dir, hdr.Name)
		if !strings.HasPrefix(target, dir+string(os.PathSeparator)) {
			return fmt.Errorf("image: illegal path in tar: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("image: mkdir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("image: mkdir %s: %w", target, err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("image: create %s: %w", target, err)
			}
			_, cerr := io.Copy(out, tr)
			out.Close()
			if cerr != nil {
				return fmt.Errorf("image: write %s: %w", target, cerr)
			}
		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return fmt.Errorf("image: symlink %s: %w", target, err)
			}
		}
	}
}
