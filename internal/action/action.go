package action

import (
	"errors"
	"fmt"
	"os"

	"dedup/internal/clone"
	"dedup/internal/dupes"
	"dedup/internal/fileid"
	"dedup/internal/fsutil"
)

// Action removes the storage dup duplicates from keeper.
type Action interface {
	Verb() string
	Apply(keeper, dup dupes.File) error
	// Reclaims reports the bytes Apply frees; a name sharing an inode with other names frees nothing.
	Reclaims(dup dupes.File) int64
	// SameDevice reports whether keeper and dup must live on one device.
	SameDevice() bool
}

func Parse(name string) (Action, error) {
	switch name {
	case "report":
		return nil, nil
	case "delete":
		return Delete{}, nil
	case "hardlink":
		return Hardlink{}, nil
	case "reflink":
		return Reflink{}, nil
	}
	return nil, fmt.Errorf("unknown action %q (want report, delete, hardlink or reflink)", name)
}

var ErrChanged = errors.New("changed since scan")

// Verify confirms f still matches what was scanned, guarding against edits between hashing and acting.
func Verify(f dupes.File) error {
	info, err := os.Lstat(f.Path)
	if err != nil {
		return err
	}
	id, err := fileid.Of(f.Path, info)
	if err != nil {
		return err
	}
	if info.Size() != f.Size || !info.ModTime().Equal(f.ModTime) || (f.ID.Known() && !id.SameObject(f.ID)) {
		return fmt.Errorf("%s: %w", f.Path, ErrChanged)
	}
	return nil
}

func soleName(f dupes.File) int64 {
	if f.ID.Links > 1 {
		return 0
	}
	return f.Size
}

type Delete struct{}

func (Delete) Verb() string                  { return "delete" }
func (Delete) Apply(_, dup dupes.File) error { return os.Remove(dup.Path) }
func (Delete) Reclaims(dup dupes.File) int64 { return soleName(dup) }
func (Delete) SameDevice() bool              { return false }

type Hardlink struct{}

func (Hardlink) Verb() string                  { return "hardlink" }
func (Hardlink) Reclaims(dup dupes.File) int64 { return soleName(dup) }
func (Hardlink) SameDevice() bool              { return true }
func (Hardlink) Apply(keeper, dup dupes.File) error {
	return fsutil.Replace(dup.Path, func(tmp string) error { return os.Link(keeper.Path, tmp) })
}

type Reflink struct{}

func (Reflink) Verb() string                  { return "reflink" }
func (Reflink) Reclaims(dup dupes.File) int64 { return dup.Size }
func (Reflink) SameDevice() bool              { return true }
func (Reflink) Apply(keeper, dup dupes.File) error {
	return clone.Share(keeper.Path, dup.Path, dup.Size)
}
