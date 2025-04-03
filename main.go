package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

// fileInfo stores the path and hash of a file.
// We don't strictly need this struct anymore as we store path directly in map,
// but keeping it doesn't hurt if we wanted to expand later.
type fileInfo struct {
	path string
	hash string
}

// Constants for byte conversion
const (
	bytesPerMegabyte = 1024.0 * 1024.0
)

func main() {
	// 1. Define and parse command-line flags
	dirPath := flag.String("dir", ".", "Directory to scan for duplicate files")
	dryRun := flag.Bool("dryrun", false, "Enable dry run mode (report actions without deleting)")
	flag.Parse()

	// Validate the directory path
	info, err := os.Stat(*dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("Error: Directory '%s' does not exist.", *dirPath)
		}
		log.Fatalf("Error accessing directory '%s': %v", *dirPath, err)
	}
	if !info.IsDir() {
		log.Fatalf("Error: '%s' is not a directory.", *dirPath)
	}

	if *dryRun {
		fmt.Println("--- DRY RUN MODE ENABLED: No files will be deleted. ---")
	}
	fmt.Printf("Scanning directory: %s\n", *dirPath)

	// 2. Data structure to store hashes and the path of the *first* file seen
	filesSeen := make(map[string]string)
	var deletedCount int = 0
	var totalBytesSaved int64 = 0 // Use int64 for potentially large sizes

	// 3. Walk the directory recursively
	err = filepath.WalkDir(*dirPath, func(path string, d fs.DirEntry, err error) error {
		// Handle errors during walking (e.g., permission issues)
		if err != nil {
			log.Printf("Warning: Error accessing path %q: %v\n", path, err)
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Skip non-regular files
		if !d.Type().IsRegular() {
			return nil
		}

		// Calculate the file's hash
		hashString, err := calculateFileHash(path)
		if err != nil {
			log.Printf("Warning: Could not calculate hash for file %q: %v. Skipping.\n", path, err)
			return nil
		}

		// Check if this hash has been seen before
		if originalPath, found := filesSeen[hashString]; found {
			// Duplicate found!

			// Get file info to find its size
			fInfo, err := d.Info()
			if err != nil {
				// This is less likely if WalkDir succeeded, but check anyway
				log.Printf("Warning: Could not get file info for %q: %v. Skipping deletion/reporting size.\n", path, err)
				return nil // Skip processing this specific duplicate
			}
			fileSize := fInfo.Size()

			if *dryRun {
				// Dry Run Mode: Report only
				fmt.Printf("[Dry Run] Would delete: '%s' (duplicate of '%s', size: %d bytes)\n", path, originalPath, fileSize)
				deletedCount++
				totalBytesSaved += fileSize
			} else {
				// Actual Deletion Mode
				fmt.Printf("Duplicate found: '%s' is same as '%s'. Deleting '%s' (size: %d bytes).\n", path, originalPath, path, fileSize)
				err := os.Remove(path)
				if err != nil {
					log.Printf("Error: Failed to delete file '%s': %v\n", path, err)
					// Don't count bytes if deletion failed
				} else {
					deletedCount++              // Increment count only if deletion succeeded
					totalBytesSaved += fileSize // Add size only if deletion succeeded
				}
			}
		} else {
			// First time seeing this hash, record it
			filesSeen[hashString] = path
		}

		return nil // Continue walking
	})

	// Check for errors encountered during the walk itself
	if err != nil {
		log.Fatalf("Error walking the path %q: %v\n", *dirPath, err)
	}

	// 4. Report results
	fmt.Println("\n--- Scan Complete ---")
	megabytesSaved := float64(totalBytesSaved) / bytesPerMegabyte

	if *dryRun {
		fmt.Printf("Total files identified for deletion: %d\n", deletedCount)
		fmt.Printf("Total disk space that would be saved: %.2f MB\n", megabytesSaved)
	} else {
		fmt.Printf("Total files actually deleted: %d\n", deletedCount)
		fmt.Printf("Total disk space saved: %.2f MB\n", megabytesSaved)
	}
}

// calculateFileHash opens a file, calculates its SHA256 hash, and returns the hex string.
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close() // Ensure file is closed

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file for hashing: %w", err)
	}

	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	return hashString, nil
}
