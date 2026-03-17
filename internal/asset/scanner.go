package asset

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ImportClass categorizes an import source.
type ImportClass int

const (
	ImportSkip    ImportClass = iota // local relative, project alias, static asset, react
	ImportNPM                       // npm dependency
	ImportPeerDep                   // wk-ui internal component dependency
)

// ScanResult holds aggregated import scan results for a directory.
type ScanResult struct {
	Deps         []string // npm package names (sorted, deduplicated)
	PeerDeps     []string // registry keys like "ui/spinner" (sorted, deduplicated)
	ScannedFiles []string // files that were scanned
}

var (
	// reBlockComment matches block comments (including multiline).
	reBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	// reSingleLineComment matches single-line comments.
	reSingleLineComment = regexp.MustCompile(`//.*$`)

	// reFromClause captures: from "pkg" / from 'pkg'
	reFromClause = regexp.MustCompile(`\bfrom\s+['"]([^'"]+)['"]`)
	// reSideEffect captures: import "pkg" / import 'pkg'
	reSideEffect = regexp.MustCompile(`^\s*import\s+['"]([^'"]+)['"]`)

	// skipSuffixes are file extensions that indicate static asset imports.
	skipSuffixes = []string{".css", ".scss", ".svg", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".woff", ".woff2"}

	// skipReactPackages are host dependencies that should not be listed as deps.
	skipReactPackages = map[string]bool{
		"react":             true,
		"react-dom":         true,
		"react/jsx-runtime": true,
	}
)

// ScanDir scans all .ts/.tsx files (excluding .d.ts) in dir and returns
// aggregated npm deps and wk-ui peer deps.
func ScanDir(dir string) (*ScanResult, error) {
	depsSet := make(map[string]bool)
	peerSet := make(map[string]bool)
	var scanned []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		name := info.Name()
		if strings.HasSuffix(name, ".d.ts") {
			return nil
		}
		if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".tsx") {
			return nil
		}

		sources, err := ScanFile(path)
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(dir, path)
		scanned = append(scanned, rel)

		for _, src := range sources {
			cls, key := classifySource(src)
			switch cls {
			case ImportNPM:
				depsSet[key] = true
			case ImportPeerDep:
				peerSet[key] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &ScanResult{
		Deps:         sortedKeys(depsSet),
		PeerDeps:     sortedKeys(peerSet),
		ScannedFiles: scanned,
	}, nil
}

// ScanFile reads a single .ts/.tsx file and returns all import sources found.
func ScanFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return scanSource(string(data)), nil
}

// scanSource extracts import sources from TypeScript/TSX source code.
func scanSource(src string) []string {
	// Strip block comments first
	cleaned := reBlockComment.ReplaceAllString(src, " ")

	seen := make(map[string]bool)
	var sources []string

	for _, line := range strings.Split(cleaned, "\n") {
		// Strip single-line comments
		line = reSingleLineComment.ReplaceAllString(line, "")

		// Try from-clause: import/export ... from "pkg"
		for _, m := range reFromClause.FindAllStringSubmatch(line, -1) {
			s := m[1]
			if !seen[s] {
				seen[s] = true
				sources = append(sources, s)
			}
		}

		// Try side-effect import: import "pkg"
		if m := reSideEffect.FindStringSubmatch(line); m != nil {
			s := m[1]
			if !seen[s] {
				seen[s] = true
				sources = append(sources, s)
			}
		}
	}

	return sources
}

// classifySource determines whether an import source is npm, peerDep, or skip.
func classifySource(src string) (ImportClass, string) {
	// Relative paths → skip
	if strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../") {
		return ImportSkip, ""
	}

	// @/shared/wk/ui/{name} or @/shared/wk/components/{name} → peerDep
	if key, ok := extractPeerDepKey(src); ok {
		return ImportPeerDep, key
	}

	// @/shared/ other (shadcn, lib, hooks, types) → skip
	if strings.HasPrefix(src, "@/shared/") {
		return ImportSkip, ""
	}

	// @/ other project alias → skip
	if strings.HasPrefix(src, "@/") {
		return ImportSkip, ""
	}

	// Static asset imports → skip
	for _, suffix := range skipSuffixes {
		if strings.HasSuffix(src, suffix) {
			return ImportSkip, ""
		}
	}

	// React host deps → skip
	if skipReactPackages[src] {
		return ImportSkip, ""
	}

	// Everything else → npm dep
	return ImportNPM, extractPackageName(src)
}

// extractPackageName extracts the npm package name from an import source.
// "@tanstack/react-table/utils" → "@tanstack/react-table" (scoped: first two segments)
// "lucide-react/icons/Arrow" → "lucide-react" (unscoped: first segment)
func extractPackageName(source string) string {
	if strings.HasPrefix(source, "@") {
		// Scoped package: @scope/name[/subpath]
		parts := strings.SplitN(source, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return source
	}
	// Unscoped: name[/subpath]
	parts := strings.SplitN(source, "/", 2)
	return parts[0]
}

// extractPeerDepKey checks if source is a wk-ui internal import and returns
// the registry key. Returns ("ui/spinner", true) for "@/shared/wk/ui/spinner/types".
func extractPeerDepKey(source string) (string, bool) {
	// Must start with @/shared/wk/
	if !strings.HasPrefix(source, "@/shared/wk/") {
		return "", false
	}

	// Remove the prefix: "ui/spinner/types" or "components/date-picker"
	rest := strings.TrimPrefix(source, "@/shared/wk/")

	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 {
		return "", false
	}

	category := parts[0]
	if category != "ui" && category != "components" {
		return "", false
	}

	// Return category/name, ignoring any sub-path
	return category + "/" + parts[1], true
}

// sortedKeys returns sorted keys from a bool map.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
