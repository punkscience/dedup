package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func Write(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func Bytes(n int, fill byte) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = fill + byte(i%7)
	}
	return b
}
