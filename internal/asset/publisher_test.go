package asset

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushToRegistry_DryRun(t *testing.T) {
	dir := t.TempDir()

	meta := &Meta{
		Name:     "spinner",
		Version:  "1.0.0",
		Category: "ui",
		Files:    []string{"spinner.tsx"},
		Deps:     []string{"class-variance-authority"},
	}

	opts := PushOptions{
		CacheDir:  dir,
		Category:  "ui",
		Name:      "spinner",
		Version:   "1.0.0",
		SourceDir: dir,
		Meta:      meta,
		DryRun:    true,
	}

	if err := PushToRegistry(opts); err != nil {
		t.Fatalf("DryRun should not error: %v", err)
	}

	// Ensure no files were written
	destDir := filepath.Join(dir, "ui", "spinner", "v1.0.0")
	if _, err := os.Stat(destDir); err == nil {
		t.Error("DryRun should not create dest dir")
	}
}

func TestPushToRegistry_VersionExists(t *testing.T) {
	dir := setupGitRepo(t)

	// Create a real file so we can commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("init"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, dir, "add", "-A")
	runGitCmd(t, dir, "commit", "-m", "init")

	// Pre-create the version directory with a file (git doesn't track empty dirs)
	destDir := filepath.Join(dir, "ui", "spinner", "v1.0.0")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	// Must commit so working tree is clean
	runGitCmd(t, dir, "add", "-A")
	runGitCmd(t, dir, "commit", "-m", "add version dir")

	opts := PushOptions{
		CacheDir:  dir,
		Category:  "ui",
		Name:      "spinner",
		Version:   "1.0.0",
		SourceDir: dir,
		Meta:      &Meta{Name: "spinner", Version: "1.0.0", Files: []string{}},
	}

	err := PushToRegistry(opts)
	if err == nil {
		t.Fatal("expected error for existing version")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want to contain 'already exists'", err.Error())
	}
}

func TestPushToRegistry_DirtyRepo(t *testing.T) {
	dir := setupGitRepo(t)

	// Create initial file and commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("init"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, dir, "add", "-A")
	runGitCmd(t, dir, "commit", "-m", "init")

	// Now make it dirty
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := PushOptions{
		CacheDir:  dir,
		Category:  "ui",
		Name:      "spinner",
		Version:   "1.0.0",
		SourceDir: dir,
		Meta:      &Meta{Name: "spinner", Version: "1.0.0", Files: []string{}},
	}

	err := PushToRegistry(opts)
	if err == nil {
		t.Fatal("expected error for dirty repo")
	}
	if !strings.Contains(err.Error(), "uncommitted changes") {
		t.Errorf("error = %q, want to contain 'uncommitted changes'", err.Error())
	}
}

func TestPushToRegistry_Success(t *testing.T) {
	cacheDir := setupGitRepo(t)

	// Create initial file and commit
	if err := os.MkdirAll(filepath.Join(cacheDir, "ui"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "README.md"), []byte("init"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, cacheDir, "add", "-A")
	runGitCmd(t, cacheDir, "commit", "-m", "init")

	// Set up source directory with a file
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "spinner.tsx"), []byte("export const Spinner = () => <div/>"), 0644); err != nil {
		t.Fatal(err)
	}

	meta := &Meta{
		Name:     "spinner",
		Version:  "1.0.0",
		Category: "ui",
		Files:    []string{"spinner.tsx"},
		Deps:     []string{"class-variance-authority"},
	}

	opts := PushOptions{
		CacheDir:  cacheDir,
		Category:  "ui",
		Name:      "spinner",
		Version:   "1.0.0",
		SourceDir: srcDir,
		Meta:      meta,
	}

	// No remote configured, so push is skipped
	err := PushToRegistry(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify files were written
	destDir := filepath.Join(cacheDir, "ui", "spinner", "v1.0.0")
	if _, err := os.Stat(filepath.Join(destDir, "spinner.tsx")); err != nil {
		t.Error("spinner.tsx should exist in dest")
	}
	if _, err := os.Stat(filepath.Join(destDir, "meta.json")); err != nil {
		t.Error("meta.json should exist in dest")
	}

	// Verify registry.json was created
	if _, err := os.Stat(filepath.Join(cacheDir, "registry.json")); err != nil {
		t.Error("registry.json should exist")
	}

	// Verify git committed
	out := runGitCmdOutput(t, cacheDir, "log", "--oneline", "-1")
	if !strings.Contains(out, "feat(ui): add spinner@1.0.0") {
		t.Errorf("commit message = %q, want to contain 'feat(ui): add spinner@1.0.0'", out)
	}
}

// setupGitRepo creates a temporary git repository for testing.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "user.name", "Test")
	return dir
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s: %v", args, string(out), err)
	}
}

func runGitCmdOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s: %v", args, string(out), err)
	}
	return string(out)
}
