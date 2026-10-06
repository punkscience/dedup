package action_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dedup/internal/action"
	"dedup/internal/dupes"
	"dedup/internal/fileid"
	"dedup/internal/scan"
	"dedup/internal/testutil"
)

func pair(t *testing.T, dir string) (dupes.File, dupes.File) {
	t.Helper()
	data := testutil.Bytes(64<<10, 'x')
	return load(t, testutil.Write(t, dir, "keep", data)), load(t, testutil.Write(t, dir, "dup", data))
}

func load(t *testing.T, path string) dupes.File {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	id, err := fileid.Of(path, info)
	if err != nil {
		t.Fatal(err)
	}
	return dupes.File{File: scan.File{Path: path, Size: info.Size(), ModTime: info.ModTime(), Info: info}, ID: id}
}

func TestDelete(t *testing.T) {
	keeper, dup := pair(t, t.TempDir())
	if err := (action.Delete{}).Apply(keeper, dup); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dup.Path); !os.IsNotExist(err) {
		t.Fatalf("dup still exists: %v", err)
	}
}

func TestHardlink(t *testing.T) {
	keeper, dup := pair(t, t.TempDir())
	if err := (action.Hardlink{}).Apply(keeper, dup); err != nil {
		t.Fatal(err)
	}
	if !load(t, dup.Path).ID.SameObject(load(t, keeper.Path).ID) {
		t.Fatal("dup is not linked to keeper")
	}
	if (action.Hardlink{}).Reclaims(load(t, dup.Path)) != 0 {
		t.Fatal("relinking a shared inode should reclaim nothing")
	}
}

// Reflink needs a copy-on-write filesystem; set DEDUP_REFLINK_DIR to a directory on Btrfs, XFS, APFS or ReFS.
func TestReflink(t *testing.T) {
	dir := os.Getenv("DEDUP_REFLINK_DIR")
	if dir == "" {
		t.Skip("DEDUP_REFLINK_DIR not set")
	}
	dir, err := os.MkdirTemp(dir, "dedup-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	keeper, dup := pair(t, dir)
	if err := (action.Reflink{}).Apply(keeper, dup); err != nil {
		if errors.Is(err, errors.ErrUnsupported) {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	got, err := os.ReadFile(dup.Path)
	if err != nil || len(got) != int(dup.Size) {
		t.Fatalf("dup unreadable after reflink: %v", err)
	}

	other := testutil.Write(t, dir, "other", testutil.Bytes(64<<10, 'y'))
	if err := (action.Reflink{}).Apply(keeper, load(t, other)); err == nil {
		t.Fatal("reflinked files with different content")
	}
}

func TestVerifyDetectsChange(t *testing.T) {
	_, dup := pair(t, t.TempDir())
	if err := action.Verify(dup); err != nil {
		t.Fatal(err)
	}
	later := dup.ModTime.Add(time.Second)
	if err := os.Chtimes(dup.Path, later, later); err != nil {
		t.Fatal(err)
	}
	if err := action.Verify(dup); !errors.Is(err, action.ErrChanged) {
		t.Fatalf("got %v, want ErrChanged", err)
	}
	if err := action.Verify(dupes.File{File: scan.File{Path: filepath.Join(t.TempDir(), "gone")}}); err == nil {
		t.Fatal("missing file verified")
	}
}
