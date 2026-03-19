package asset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

type FileDiff struct {
	File   string
	Status string // "modified", "added", "deleted", "unchanged"
	Diff   string // unified diff output (empty if unchanged)
}

// DiffAsset compares local asset files against cached (original) versions.
// localDir is the project-side directory, cacheDir is the asset source directory.
func DiffAsset(localDir, cacheDir string, files []string) ([]FileDiff, error) {
	var diffs []FileDiff

	for _, f := range files {
		localPath := filepath.Join(localDir, f)
		cachePath := filepath.Join(cacheDir, f)

		localContent, localErr := os.ReadFile(localPath)
		cacheContent, cacheErr := os.ReadFile(cachePath)

		switch {
		case localErr != nil && cacheErr != nil:
			continue
		case localErr != nil:
			diffs = append(diffs, FileDiff{
				File:   f,
				Status: "deleted",
				Diff:   formatDeletedDiff(f, string(cacheContent)),
			})
		case cacheErr != nil:
			diffs = append(diffs, FileDiff{
				File:   f,
				Status: "added",
				Diff:   formatAddedDiff(f, string(localContent)),
			})
		default:
			if string(localContent) == string(cacheContent) {
				diffs = append(diffs, FileDiff{
					File:   f,
					Status: "unchanged",
				})
			} else {
				diffs = append(diffs, FileDiff{
					File:   f,
					Status: "modified",
					Diff:   unifiedDiff(f, string(cacheContent), string(localContent)),
				})
			}
		}
	}

	return diffs, nil
}

// HasLocalChanges returns true if any file in the asset has been modified locally.
func HasLocalChanges(localDir, cacheDir string, files []string) bool {
	for _, f := range files {
		localPath := filepath.Join(localDir, f)
		cachePath := filepath.Join(cacheDir, f)

		local, lerr := os.ReadFile(localPath)
		cache, cerr := os.ReadFile(cachePath)

		if lerr != nil || cerr != nil {
			return true
		}
		if string(local) != string(cache) {
			return true
		}
	}
	return false
}

// unifiedDiff produces a standard unified diff using the Myers algorithm (via go-difflib).
func unifiedDiff(filename, original, modified string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(original),
		B:        difflib.SplitLines(modified),
		FromFile: "a/" + filename,
		ToFile:   "b/" + filename,
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}

func formatDeletedDiff(filename, content string) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("--- a/%s\n", filename))
	buf.WriteString("+++ /dev/null\n")
	for _, line := range splitLines(content) {
		buf.WriteString("-" + line + "\n")
	}
	return buf.String()
}

func formatAddedDiff(filename, content string) string {
	var buf strings.Builder
	buf.WriteString("--- /dev/null\n")
	buf.WriteString(fmt.Sprintf("+++ b/%s\n", filename))
	for _, line := range splitLines(content) {
		buf.WriteString("+" + line + "\n")
	}
	return buf.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
