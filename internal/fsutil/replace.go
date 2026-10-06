package fsutil

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
)

// Replace atomically swaps dst for a file that create writes at a temporary path beside it.
func Replace(dst string, create func(tmp string) error) error {
	tmp, err := tempName(dst)
	if err != nil {
		return err
	}
	if err := create(tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// CopyMeta gives tmp the mode and timestamps of src.
func CopyMeta(src os.FileInfo, tmp string) error {
	if err := os.Chmod(tmp, src.Mode().Perm()); err != nil {
		return err
	}
	return os.Chtimes(tmp, src.ModTime(), src.ModTime())
}

func tempName(dst string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(dst), ".dedup-"+hex.EncodeToString(b)), nil
}
