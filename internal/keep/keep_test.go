package keep_test

import (
	"path/filepath"
	"testing"
	"time"

	"dedup/internal/dupes"
	"dedup/internal/keep"
	"dedup/internal/scan"
)

func file(path string, age time.Duration) dupes.File {
	return dupes.File{File: scan.File{Path: path, ModTime: time.Unix(1e9, 0).Add(-age)}}
}

func TestSplit(t *testing.T) {
	g := dupes.Group{
		file(filepath.FromSlash("/x/copies/long-name"), 1*time.Hour),
		file(filepath.FromSlash("/x/b"), 2*time.Hour),
		file(filepath.FromSlash("/x/photos/a"), 0),
	}
	cases := []struct {
		name string
		sel  keep.Selector
		want string
	}{
		{"oldest", keep.Selector{Rule: keep.Oldest}, "/x/b"},
		{"newest", keep.Selector{Rule: keep.Newest}, "/x/photos/a"},
		{"shortest", keep.Selector{Rule: keep.Shortest}, "/x/b"},
		{"first", keep.Selector{Rule: keep.First}, "/x/b"},
		{"prefer beats rule", keep.Selector{Rule: keep.Oldest, Prefer: []string{filepath.FromSlash("/x/photos")}}, "/x/photos/a"},
		{"prefer is not a string prefix", keep.Selector{Rule: keep.Newest, Prefer: []string{filepath.FromSlash("/x/cop")}}, "/x/photos/a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			keeper, dups := c.sel.Split(g)
			if keeper.Path != filepath.FromSlash(c.want) {
				t.Errorf("kept %s, want %s", keeper.Path, c.want)
			}
			if len(dups) != 2 {
				t.Errorf("got %d dups, want 2", len(dups))
			}
		})
	}
}
