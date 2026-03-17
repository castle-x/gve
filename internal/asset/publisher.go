package asset

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PushOptions configures a push-to-registry operation.
type PushOptions struct {
	CacheDir  string // root of the registry cache (e.g. ~/.gve/cache/ui)
	Category  string // "ui" or "components"
	Name      string // bare asset name (e.g. "spinner")
	Version   string // semver without "v" prefix (e.g. "1.2.0")
	SourceDir string // local directory containing the component files
	Meta      *Meta  // pre-built meta.json content
	CommitMsg string // git commit message
	DryRun    bool   // if true, print what would happen but don't modify anything
}

// PushToRegistry publishes a UI component to the registry cache and pushes to remote.
// It performs: git pull → clean check → version check → copy files → write meta →
// rebuild registry.json → git add → commit → push.
// On failure after file writes, it attempts to roll back.
func PushToRegistry(opts PushOptions) error {
	registryKey := opts.Category + "/" + opts.Name
	versionDir := "v" + opts.Version
	destDir := filepath.Join(opts.CacheDir, opts.Category, opts.Name, versionDir)
	registryPath := filepath.Join(opts.CacheDir, "registry.json")

	if opts.DryRun {
		fmt.Printf("[dry-run] Would publish %s@%s\n", registryKey, opts.Version)
		fmt.Printf("[dry-run] Source: %s\n", opts.SourceDir)
		fmt.Printf("[dry-run] Dest: %s\n", destDir)
		fmt.Printf("[dry-run] Meta: %s\n", formatMetaPreview(opts.Meta))
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

	// Phase 4: copy files
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	if err := CopyAsset(opts.SourceDir, destDir, opts.Meta.Files); err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("copy files: %w", err)
	}

	// Write meta.json
	metaData, err := json.MarshalIndent(opts.Meta, "", "  ")
	if err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("marshal meta: %w", err)
	}
	metaPath := filepath.Join(destDir, "meta.json")
	if err := os.WriteFile(metaPath, append(metaData, '\n'), 0644); err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("write meta.json: %w", err)
	}

	// Phase 5: rebuild registry.json
	reg, warnings, err := BuildRegistryV2(opts.CacheDir)
	if err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("rebuild registry: %w", err)
	}
	for _, w := range warnings {
		fmt.Println(w)
	}
	if err := WriteRegistryV2(reg, registryPath); err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("write registry.json: %w", err)
	}

	// Phase 6: git add
	relDest, _ := filepath.Rel(opts.CacheDir, destDir)
	if _, err := runGit(opts.CacheDir, "add", relDest, "registry.json"); err != nil {
		os.RemoveAll(destDir) // rollback
		return fmt.Errorf("git add: %w", err)
	}

	// Phase 7: git commit
	commitMsg := opts.CommitMsg
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("feat(%s): add %s@%s", opts.Category, opts.Name, opts.Version)
	}
	if _, err := runGit(opts.CacheDir, "commit", "-m", commitMsg); err != nil {
		// rollback: reset the commit staging and remove dest
		runGit(opts.CacheDir, "reset", "HEAD")
		os.RemoveAll(destDir)
		return fmt.Errorf("git commit: %w", err)
	}

	// Phase 8: git push — do NOT rollback commit on push failure
	if hasRemote(opts.CacheDir) {
		if _, err := runGit(opts.CacheDir, "push", "origin", "HEAD"); err != nil {
			return fmt.Errorf("git push failed — commit is local, retry with: cd %s && git push origin HEAD\n%w",
				opts.CacheDir, err)
		}
	}

	return nil
}

// hasRemote checks if the git repo has any remote configured.
func hasRemote(dir string) bool {
	out, err := runGit(dir, "remote")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// runGit executes a git command in the given directory and returns combined output.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
	}
	return string(out), nil
}

// formatMetaPreview returns a brief summary of a Meta for dry-run output.
func formatMetaPreview(m *Meta) string {
	if m == nil {
		return "<nil>"
	}
	parts := []string{
		fmt.Sprintf("name=%s", m.Name),
		fmt.Sprintf("version=%s", m.Version),
		fmt.Sprintf("files=%d", len(m.Files)),
	}
	if len(m.Deps) > 0 {
		parts = append(parts, fmt.Sprintf("deps=%v", m.Deps))
	}
	if len(m.PeerDeps) > 0 {
		parts = append(parts, fmt.Sprintf("peerDeps=%v", m.PeerDeps))
	}
	return strings.Join(parts, ", ")
}
