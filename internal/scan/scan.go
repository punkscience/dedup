package scan

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"time"
)

type File struct {
	Path    string
	Size    int64
	ModTime time.Time
	Info    fs.FileInfo
}

// Walk returns every regular file under root of at least minSize bytes.
// Unreadable paths are reported to warn and skipped.
func Walk(root string, minSize int64, warn func(error)) ([]File, error) {
	var files []File
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			warn(err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			warn(fmt.Errorf("stat %s: %w", path, err))
			return nil
		}
		if info.Size() < minSize {
			return nil
		}
		files = append(files, File{Path: path, Size: info.Size(), ModTime: info.ModTime(), Info: info})
		return nil
	})
	return files, err
}
