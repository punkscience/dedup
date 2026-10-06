package keep

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"dedup/internal/dupes"
)

type Rule string

const (
	Oldest   Rule = "oldest"
	Newest   Rule = "newest"
	Shortest Rule = "shortest"
	First    Rule = "first"
)

func ParseRule(s string) (Rule, error) {
	switch r := Rule(s); r {
	case Oldest, Newest, Shortest, First:
		return r, nil
	}
	return "", fmt.Errorf("unknown keep rule %q (want oldest, newest, shortest or first)", s)
}

// Selector picks the copy to keep: files under a Prefer directory win, then Rule, then path order.
type Selector struct {
	Rule   Rule
	Prefer []string
}

func (s Selector) Split(g dupes.Group) (dupes.File, []dupes.File) {
	sorted := slices.Clone(g)
	slices.SortStableFunc(sorted, s.compare)
	return sorted[0], sorted[1:]
}

func (s Selector) compare(a, b dupes.File) int {
	return cmp.Or(
		boolFirst(s.preferred(a.Path), s.preferred(b.Path)),
		s.byRule(a, b),
		cmp.Compare(a.Path, b.Path),
	)
}

func (s Selector) byRule(a, b dupes.File) int {
	switch s.Rule {
	case Oldest:
		return a.ModTime.Compare(b.ModTime)
	case Newest:
		return b.ModTime.Compare(a.ModTime)
	case Shortest:
		return cmp.Compare(len(a.Path), len(b.Path))
	}
	return 0
}

func (s Selector) preferred(path string) bool {
	for _, dir := range s.Prefer {
		rel, err := filepath.Rel(dir, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func boolFirst(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return -1
	}
	return 1
}
