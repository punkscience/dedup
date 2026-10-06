package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"dedup/internal/action"
	"dedup/internal/dupes"
	"dedup/internal/keep"
	"dedup/internal/scan"
)

type config struct {
	dir     string
	act     action.Action
	verb    string
	dryRun  bool
	keeper  keep.Selector
	workers int
	minSize int64
}

var pastTense = map[string]string{"delete": "deleted", "hardlink": "hardlinked", "reflink": "reflinked"}

type summary struct {
	groups, dups, failed int
	bytes                int64
}

func main() {
	log.SetFlags(0)
	cfg, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	if err := run(cfg, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func parseFlags() (config, error) {
	var cfg config
	var prefer []string
	actName := flag.String("action", "report", "report, delete, hardlink or reflink")
	keepRule := flag.String("keep", "oldest", "copy to keep: oldest, newest, shortest or first")
	flag.StringVar(&cfg.dir, "dir", ".", "directory to scan")
	flag.BoolVar(&cfg.dryRun, "dryrun", false, "show what -action would do without doing it")
	flag.IntVar(&cfg.workers, "workers", runtime.NumCPU(), "parallel readers (use 1-2 on spinning disks)")
	flag.Int64Var(&cfg.minSize, "min-size", 1, "ignore files smaller than this many bytes")
	flag.Func("prefer", "keep copies under this directory first (repeatable)", func(s string) error {
		prefer = append(prefer, s)
		return nil
	})
	flag.Parse()

	act, err := action.Parse(*actName)
	if err != nil {
		return cfg, err
	}
	rule, err := keep.ParseRule(*keepRule)
	if err != nil {
		return cfg, err
	}
	cfg.act, cfg.verb, cfg.keeper = act, *actName, keep.Selector{Rule: rule, Prefer: prefer}

	info, err := os.Stat(cfg.dir)
	if err != nil {
		return cfg, err
	}
	if !info.IsDir() {
		return cfg, fmt.Errorf("%s is not a directory", cfg.dir)
	}
	return cfg, nil
}

func run(cfg config, out io.Writer) error {
	warn := func(err error) { log.Printf("warning: %v", err) }

	files, err := scan.Walk(cfg.dir, cfg.minSize, warn)
	if err != nil {
		return err
	}
	groups := dupes.Find(files, dupes.Options{Workers: cfg.workers, Warn: warn})
	if cfg.act != nil && cfg.act.SameDevice() {
		groups = dupes.SplitByDevice(groups)
	}

	var s summary
	for _, g := range groups {
		s.groups++
		keeper, dups := cfg.keeper.Split(g)
		fmt.Fprintf(out, "\n%d copies, %s each\n  keep  %s\n", len(g), human(keeper.Size), keeper.Path)
		for _, dup := range dups {
			s.apply(cfg, keeper, dup, out)
		}
	}
	s.print(cfg, len(files), out)
	return nil
}

func (s *summary) apply(cfg config, keeper, dup dupes.File, out io.Writer) {
	if cfg.act == nil || cfg.dryRun {
		fmt.Fprintf(out, "  dup   %s\n", dup.Path)
		s.dups++
		s.bytes += reclaimable(cfg.act, dup)
		return
	}
	err := errors.Join(action.Verify(keeper), action.Verify(dup))
	if err == nil {
		err = cfg.act.Apply(keeper, dup)
	}
	if err != nil {
		fmt.Fprintf(out, "  FAIL  %s: %v\n", dup.Path, err)
		s.failed++
		return
	}
	fmt.Fprintf(out, "  dup   %s (%s)\n", dup.Path, pastTense[cfg.verb])
	s.dups++
	s.bytes += cfg.act.Reclaims(dup)
}

func (s summary) print(cfg config, scanned int, out io.Writer) {
	verb := "reclaimable"
	switch {
	case cfg.act != nil && cfg.dryRun:
		verb = "would be reclaimed by " + cfg.verb
	case cfg.act != nil:
		verb = "reclaimed by " + cfg.verb
	}
	fmt.Fprintf(out, "\nScanned %d files: %d duplicate groups, %d duplicates, %s %s\n",
		scanned, s.groups, s.dups, human(s.bytes), verb)
	if s.failed > 0 {
		fmt.Fprintf(out, "%d duplicates failed\n", s.failed)
	}
}

func reclaimable(act action.Action, dup dupes.File) int64 {
	if act == nil {
		return action.Delete{}.Reclaims(dup)
	}
	return act.Reclaims(dup)
}

func human(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
