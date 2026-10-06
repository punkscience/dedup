package dupes

import (
	"cmp"
	"fmt"
	"slices"
	"sync"

	"dedup/internal/fileid"
	"dedup/internal/scan"
)

type File struct {
	scan.File
	ID fileid.ID
}

// Group holds files with identical content, sorted by path.
type Group []File

type Options struct {
	Workers int
	Warn    func(error)
}

// Find narrows files to groups of identical content, reading as little as possible:
// size, then file identity, then the first and last 16 KiB, then the full content.
func Find(files []scan.File, opts Options) []Group {
	groups := bySize(files)
	groups = withIdentity(groups, opts)
	groups = regroup(groups, opts, func(Group) bool { return true }, edgeSum)
	groups = regroup(groups, opts, func(g Group) bool { return !coversWholeFile(g[0].Size) }, fullSum)
	sortGroups(groups)
	return groups
}

// SplitByDevice splits groups so that every file in a group lives on the same device.
func SplitByDevice(groups []Group) []Group {
	var out []Group
	for _, g := range groups {
		byDev := map[uint64]Group{}
		for _, f := range g {
			byDev[f.ID.Dev] = append(byDev[f.ID.Dev], f)
		}
		for _, sub := range byDev {
			if len(sub) > 1 {
				out = append(out, sub)
			}
		}
	}
	sortGroups(out)
	return out
}

func bySize(files []scan.File) []Group {
	m := map[int64]Group{}
	for _, f := range files {
		m[f.Size] = append(m[f.Size], File{File: f})
	}
	var groups []Group
	for _, g := range m {
		if len(g) > 1 {
			groups = append(groups, g)
		}
	}
	return groups
}

func withIdentity(groups []Group, opts Options) []Group {
	var out []Group
	for _, g := range groups {
		var kept Group
		for _, f := range g {
			id, err := fileid.Of(f.Path, f.Info)
			if err != nil {
				opts.Warn(fmt.Errorf("identify %s: %w", f.Path, err))
				continue
			}
			f.ID = id
			if !slices.ContainsFunc(kept, func(k File) bool { return k.ID.SameObject(id) }) {
				kept = append(kept, f)
			}
		}
		if len(kept) > 1 {
			out = append(out, kept)
		}
	}
	return out
}

type job struct {
	group int
	file  File
}

type keyed struct {
	group int
	sum   digest
}

// regroup splits each selected group by the digest sum produces, in parallel, dropping singletons.
func regroup(groups []Group, opts Options, selected func(Group) bool, sum func(File) (digest, error)) []Group {
	var out []Group
	var jobs []job
	for i, g := range groups {
		if !selected(g) {
			out = append(out, g)
			continue
		}
		for _, f := range g {
			jobs = append(jobs, job{group: i, file: f})
		}
	}
	slices.SortFunc(jobs, func(a, b job) int { return cmp.Compare(a.file.ID.Ino, b.file.ID.Ino) })

	results := make(map[keyed]Group)
	var mu sync.Mutex
	queue := make(chan job)
	var wg sync.WaitGroup
	for range max(opts.Workers, 1) {
		wg.Go(func() {
			for j := range queue {
				d, err := sum(j.file)
				if err != nil {
					opts.Warn(fmt.Errorf("hash %s: %w", j.file.Path, err))
					continue
				}
				k := keyed{group: j.group, sum: d}
				mu.Lock()
				results[k] = append(results[k], j.file)
				mu.Unlock()
			}
		})
	}
	for _, j := range jobs {
		queue <- j
	}
	close(queue)
	wg.Wait()

	for _, g := range results {
		if len(g) > 1 {
			out = append(out, g)
		}
	}
	return out
}

func sortGroups(groups []Group) {
	for _, g := range groups {
		slices.SortFunc(g, func(a, b File) int { return cmp.Compare(a.Path, b.Path) })
	}
	slices.SortFunc(groups, func(a, b Group) int {
		return cmp.Or(cmp.Compare(b[0].Size, a[0].Size), cmp.Compare(a[0].Path, b[0].Path))
	})
}
