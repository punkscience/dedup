package dupes_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"dedup/internal/dupes"
	"dedup/internal/scan"
	"dedup/internal/testutil"
)

func find(t *testing.T, dir string) [][]string {
	t.Helper()
	files, err := scan.Walk(dir, 1, func(err error) { t.Error(err) })
	if err != nil {
		t.Fatal(err)
	}
	var out [][]string
	for _, g := range dupes.Find(files, dupes.Options{Workers: 4, Warn: func(err error) { t.Error(err) }}) {
		var names []string
		for _, f := range g {
			names = append(names, filepath.Base(f.Path))
		}
		out = append(out, names)
	}
	return out
}

func TestFindGroupsIdenticalContent(t *testing.T) {
	dir := t.TempDir()
	big := testutil.Bytes(100<<10, 'a')
	middleDiffers := slices.Clone(big)
	middleDiffers[50<<10] ^= 0xff

	testutil.Write(t, dir, "a", big)
	testutil.Write(t, dir, "sub/b", big)
	testutil.Write(t, dir, "c", middleDiffers)
	testutil.Write(t, dir, "d", big[:len(big)-1])
	testutil.Write(t, dir, "small1", []byte("hello"))
	testutil.Write(t, dir, "small2", []byte("hello"))
	testutil.Write(t, dir, "small3", []byte("world"))
	testutil.Write(t, dir, "empty1", nil)
	testutil.Write(t, dir, "empty2", nil)

	got := find(t, dir)
	want := [][]string{{"a", "b"}, {"small1", "small2"}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFindCollapsesHardlinks(t *testing.T) {
	dir := t.TempDir()
	a := testutil.Write(t, dir, "a", []byte("same"))
	if err := os.Link(a, filepath.Join(dir, "a-link")); err != nil {
		t.Skip("hardlinks unsupported:", err)
	}
	if got := find(t, dir); len(got) != 0 {
		t.Fatalf("hardlinks reported as duplicates: %v", got)
	}

	testutil.Write(t, dir, "b", []byte("same"))
	got := find(t, dir)
	if len(got) != 1 || len(got[0]) != 2 {
		t.Fatalf("want one pair, got %v", got)
	}
}
