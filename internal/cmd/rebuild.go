package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// needsRebuild checks if the binary is older than any source file.
// Returns true if the binary doesn't exist or any source is newer.
func needsRebuild(projectDir, binaryPath string) (bool, error) {
	info, err := os.Stat(binaryPath)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	binaryTime := info.ModTime()

	sourceDirs := []string{
		filepath.Join(projectDir, "cmd"),
		filepath.Join(projectDir, "internal"),
		filepath.Join(projectDir, "site", "src"),
	}

	singleFiles := []string{
		filepath.Join(projectDir, "go.mod"),
		filepath.Join(projectDir, "go.sum"),
		filepath.Join(projectDir, "site", "package.json"),
		filepath.Join(projectDir, "site", "vite.config.ts"),
		filepath.Join(projectDir, "site", "tsconfig.json"),
		filepath.Join(projectDir, "site", "index.html"),
	}

	for _, f := range singleFiles {
		if newerThan(f, binaryTime) {
			return true, nil
		}
	}

	for _, dir := range sourceDirs {
		newer, err := dirHasNewerFile(dir, binaryTime)
		if err != nil {
			continue
		}
		if newer {
			return true, nil
		}
	}

	return false, nil
}

func newerThan(path string, ref time.Time) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.ModTime().After(ref)
}

func dirHasNewerFile(dir string, ref time.Time) (bool, error) {
	found := false
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "node_modules" || base == "dist" || base == ".git" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(ref) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found, err
}
