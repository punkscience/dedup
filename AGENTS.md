# AGENTS.md

Project-specific context for AI coding agents operating in this repository.

## Project overview

`dedup` is a portable Go CLI that finds files with identical content under a directory and, optionally, reclaims the space by deleting, hardlinking or reflinking the duplicates. It reads as little as possible and keeps no on-disk index.

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go (`go 1.26.0` in `go.mod`) |
| Dependencies | `golang.org/x/sys` (reflink and Windows file identity) |
| Testing | `go test` |
| Build | `go build` |

## Commands

- **Build:** `go build -o dedup .`
- **Test:** `go test -race ./...`
- **Reflink test:** `DEDUP_REFLINK_DIR=<dir on btrfs/xfs/apfs/refs> go test ./internal/action`
- **Cross-check platforms:** `GOOS=windows go vet ./...`, `GOOS=darwin go vet ./...`
- **Format:** `gofmt -w .`

If `/tmp` (tmpfs) fills during builds, set `GOTMPDIR` and `TMPDIR` to a directory on disk.

## Usage

| Flag | Default | Meaning |
|------|---------|---------|
| `-dir` | `.` | Directory to scan |
| `-action` | `report` | `report`, `delete`, `hardlink` or `reflink` |
| `-dryrun` | `false` | Show what `-action` would do |
| `-keep` | `oldest` | Copy to keep: `oldest`, `newest`, `shortest`, `first` (path order) |
| `-prefer` | — | Keep copies under this directory first; repeatable |
| `-workers` | CPU count | Parallel readers; use 1–2 on spinning disks |
| `-min-size` | `1` | Ignore smaller files (empty files skipped by default) |

## Pipeline

1. `scan` walks the tree (unreadable paths warn and skip).
2. `dupes` groups by size, collapses hardlinks by device+inode, hashes the first and last 16 KiB, then fully hashes (SHA-256) only the survivors. Files ≤ 32 KiB are fully hashed in the first pass.
3. `keep` picks the surviving copy.
4. `action` re-stats both files (size, mtime, identity) before acting.

## Project structure

- `main.go` — flags, wiring, output
- `internal/scan` — directory walk
- `internal/fileid` — device/inode/link-count per OS (`unix`, `windows`, fallback)
- `internal/dupes` — narrowing pipeline and parallel hashing
- `internal/keep` — keeper selection
- `internal/action` — delete / hardlink / reflink and pre-action verification
- `internal/clone` — reflink per OS: Linux `FIDEDUPERANGE` (in place, kernel-verified), macOS `clonefile`, Windows ReFS `FSCTL_DUPLICATE_EXTENTS_TO_FILE`
- `internal/fsutil` — atomic replace via temp file + rename
- `internal/testutil` — test helpers

## Conventions

- Punk Science coding standard; SOLID. New actions implement `action.Action`; new platforms add a build-tagged file.
- Standard `gofmt`.

## Behaviour notes

- Default action is `report`; nothing is modified unless `-action` is given.
- Hardlink and reflink split groups by device so the keeper is always on the same volume.
- Hardlinking replaces the duplicate's inode, so its own permissions and mtime are lost. Reflink keeps them.
- Reclaimed bytes count 0 for a name whose inode has other links.
- macOS and Windows reflink paths compile but have not been run on real hardware.

## Notes

- Remote: `https://github.com/punkscience/dedup`
