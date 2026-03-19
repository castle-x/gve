package asset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// APIPushOptions configures an API push-to-registry operation.
type APIPushOptions struct {
	CacheDir  string // root of the API registry cache (e.g. ~/.gve/cache/api)
	Project   string // e.g. "ai-worker"
	Resource  string // e.g. "task"
	Version   string // e.g. "v2"
	SourceDir string // local directory containing .thrift files
	DryRun    bool
}

// PushAPIToRegistry publishes thrift files to the API registry cache and pushes to remote.
// Flow: dry-run check → git pull → clean check → version check → copy .thrift files →
// rebuild registry.json → git add → commit → push.
// On failure after file writes, it attempts to roll back.
func PushAPIToRegistry(opts APIPushOptions) error {
	registryKey := opts.Project + "/" + opts.Resource
	destDir := filepath.Join(opts.CacheDir, opts.Project, opts.Resource, opts.Version)
	registryPath := filepath.Join(opts.CacheDir, "registry.json")

	if opts.DryRun {
		fmt.Printf("[dry-run] Would publish API %s@%s\n", registryKey, opts.Version)
		fmt.Printf("[dry-run] Source: %s\n", opts.SourceDir)
		fmt.Printf("[dry-run] Dest: %s\n", destDir)

		// List .thrift files that would be copied
		thriftFiles, _ := collectThriftFiles(opts.SourceDir)
		fmt.Printf("[dry-run] Thrift files: %s\n", strings.Join(thriftFiles, ", "))
		return nil
	}

	// Phase 1: git pull --ff-only (skip if no remote configured)
	if hasRemote(opts.CacheDir) {
		if _, err := runGit(opts.CacheDir, "pull", "--ff-only"); err != nil {
			return fmt.Errorf("git pull failed (try manual rebase): %w", err)
		}
	}

	// Phase 2: check working tree is clean
	status, err := runGit(opts.CacheDir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return fmt.Errorf("registry cache has uncommitted changes — please commit or stash first:\n%s", status)
	}

	// Phase 3: check version doesn't already exist
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("version %s of %s already exists at %s", opts.Version, registryKey, destDir)
	}

	// Phase 4: collect .thrift files from source
	thriftFiles, err := collectThriftFiles(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("collect thrift files: %w", err)
	}
	if len(thriftFiles) == 0 {
		return fmt.Errorf("no .thrift files found in %s", opts.SourceDir)
	}

	// Phase 5: copy .thrift files to cache
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	if err := CopyAsset(opts.SourceDir, destDir, thriftFiles); err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("copy thrift files: %w", err)
	}

	// Phase 6: rebuild registry.json
	reg, err := BuildAPIRegistry(opts.CacheDir)
	if err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("rebuild api registry: %w", err)
	}
	if err := WriteAPIRegistry(reg, registryPath); err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("write api registry.json: %w", err)
	}

	// Phase 7: git add
	relDest, _ := filepath.Rel(opts.CacheDir, destDir)
	if _, err := runGit(opts.CacheDir, "add", relDest, "registry.json"); err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("git add: %w", err)
	}

	// Phase 8: git commit
	commitMsg := fmt.Sprintf("feat(api): add %s@%s", registryKey, opts.Version)
	if _, err := runGit(opts.CacheDir, "commit", "-m", commitMsg); err != nil {
		// rollback: reset the commit staging and remove dest
		runGit(opts.CacheDir, "reset", "HEAD")
		os.RemoveAll(destDir)
		return fmt.Errorf("git commit: %w", err)
	}

	// Phase 9: git push — do NOT rollback commit on push failure
	if hasRemote(opts.CacheDir) {
		if _, err := runGit(opts.CacheDir, "push", "origin", "HEAD"); err != nil {
			return fmt.Errorf("git push failed — commit is local, retry with: cd %s && git push origin HEAD\n%w",
				opts.CacheDir, err)
		}
	}

	return nil
}

// collectThriftFiles lists all .thrift files in a directory (non-recursive).
func collectThriftFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".thrift" {
			files = append(files, e.Name())
		}
	}
	return files, nil
}
