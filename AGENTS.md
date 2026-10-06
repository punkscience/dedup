# AGENTS.md

Project-specific context for AI coding agents operating in this repository.

## Project overview

`dedup` is a single-file Go CLI that recursively scans a directory, hashes every regular file with SHA-256, and deletes duplicates — keeping the first file encountered for each hash. A dry-run mode reports what would be deleted and the space saved.

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go (`go 1.21.6` in `go.mod`) |
| Package manager | Go modules (stdlib only, no `go.sum`) |
| Testing | None yet |
| Build | `go build` |

## Commands

- **Build:** `go build -o dedup .`
- **Run:** `go run . -dir <path> [-dryrun]`
- **Test:** `go test ./...` (no tests exist yet)
- **Format:** `gofmt -w .`
- **Vet:** `go vet ./...`

## Usage

| Flag | Default | Meaning |
|------|---------|---------|
| `-dir` | `.` | Directory to scan recursively |
| `-dryrun` | `false` | Report deletions without removing files |

## Project structure

- `main.go` — entire program: flag parsing, `filepath.WalkDir` traversal, `calculateFileHash`, summary report
- `go.mod` — module `dedup`

## Conventions

- **Code style:** standard `gofmt`.
- **Module layout:** flat, single `main` package.
- **Coding standard:** Punk Science coding standard; SOLID.

## Behaviour notes

- **Destructive by default.** Without `-dryrun`, duplicates are deleted immediately via `os.Remove`. Always use `-dryrun` or a throwaway directory when testing.
- **Which copy survives** depends on `WalkDir` lexical order — the first path seen for a hash is kept; later ones are deleted.
- The `WalkDir` callback returns the error on access failures, which aborts the whole walk despite logging it as a "Warning".
- `fileInfo` struct is unused.
- Symlinks and other non-regular files are skipped.

## Notes

- Remote: `https://github.com/punkscience/dedup`
- No `.gitignore`; a built `dedup` binary in the root would be untracked.
