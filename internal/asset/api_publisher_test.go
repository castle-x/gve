package asset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushAPIToRegistry_DryRun(t *testing.T) {
	cacheDir := t.TempDir()
	sourceDir := t.TempDir()

	// Create a .thrift source file
	os.WriteFile(filepath.Join(sourceDir, "user.thrift"), []byte("service User {}"), 0644)

	opts := APIPushOptions{
		CacheDir:  cacheDir,
		Project:   "my-project",
		Resource:  "user",
		Version:   "v1",
		SourceDir: sourceDir,
		DryRun:    true,
	}

	if err := PushAPIToRegistry(opts); err != nil {
		t.Fatalf("dry-run should succeed: %v", err)
	}

	// Verify no files were written
	destDir := filepath.Join(cacheDir, "my-project", "user", "v1")
	if _, err := os.Stat(destDir); err == nil {
		t.Error("dry-run should not create destination directory")
	}
}

func TestPushAPIToRegistry_VersionExists(t *testing.T) {
	cacheDir := setupGitRepo(t)

	// Create initial commit
	os.WriteFile(filepath.Join(cacheDir, "README.md"), []byte("init"), 0644)
	runGitCmd(t, cacheDir, "add", "-A")
	runGitCmd(t, cacheDir, "commit", "-m", "init")

	// Pre-create the version directory
	destDir := filepath.Join(cacheDir, "my-project", "user", "v1")
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(destDir, "user.thrift"), []byte("service User {}"), 0644)
	runGitCmd(t, cacheDir, "add", "-A")
	runGitCmd(t, cacheDir, "commit", "-m", "existing")

	sourceDir := t.TempDir()
	os.WriteFile(filepath.Join(sourceDir, "user.thrift"), []byte("service User { string hello() }"), 0644)

	opts := APIPushOptions{
		CacheDir:  cacheDir,
		Project:   "my-project",
		Resource:  "user",
		Version:   "v1",
		SourceDir: sourceDir,
		DryRun:    false,
	}

	err := PushAPIToRegistry(opts)
	if err == nil {
		t.Fatal("should fail when version already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, should mention 'already exists'", err.Error())
	}
}

func TestPushAPIToRegistry_DirtyWorkTree(t *testing.T) {
	cacheDir := setupGitRepo(t)

	// Create initial commit
	os.WriteFile(filepath.Join(cacheDir, "README.md"), []byte("init"), 0644)
	runGitCmd(t, cacheDir, "add", "-A")
	runGitCmd(t, cacheDir, "commit", "-m", "init")

	// Make the working tree dirty
	os.WriteFile(filepath.Join(cacheDir, "dirty.txt"), []byte("uncommitted"), 0644)

	sourceDir := t.TempDir()
	os.WriteFile(filepath.Join(sourceDir, "user.thrift"), []byte("service User {}"), 0644)

	opts := APIPushOptions{
		CacheDir:  cacheDir,
		Project:   "my-project",
		Resource:  "user",
		Version:   "v1",
		SourceDir: sourceDir,
		DryRun:    false,
	}

	err := PushAPIToRegistry(opts)
	if err == nil {
		t.Fatal("should fail when working tree is dirty")
	}
	if !strings.Contains(err.Error(), "uncommitted changes") {
		t.Errorf("error = %q, should mention 'uncommitted changes'", err.Error())
	}
}

func TestPushAPIToRegistry_Success(t *testing.T) {
	cacheDir := setupGitRepo(t)

	// Create initial commit
	os.WriteFile(filepath.Join(cacheDir, "README.md"), []byte("init"), 0644)
	runGitCmd(t, cacheDir, "add", "-A")
	runGitCmd(t, cacheDir, "commit", "-m", "init")

	// Create source thrift files
	sourceDir := t.TempDir()
	os.WriteFile(filepath.Join(sourceDir, "user.thrift"), []byte("service User {}"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "common.thrift"), []byte("struct Pagination {}"), 0644)

	opts := APIPushOptions{
		CacheDir:  cacheDir,
		Project:   "my-project",
		Resource:  "user",
		Version:   "v1",
		SourceDir: sourceDir,
		DryRun:    false,
	}

	if err := PushAPIToRegistry(opts); err != nil {
		t.Fatalf("push should succeed: %v", err)
	}

	// Verify files were copied
	destDir := filepath.Join(cacheDir, "my-project", "user", "v1")
	for _, f := range []string{"user.thrift", "common.thrift"} {
		if _, err := os.Stat(filepath.Join(destDir, f)); err != nil {
			t.Errorf("expected file %s not found: %v", f, err)
		}
	}

	// Verify registry.json was created
	regPath := filepath.Join(cacheDir, "registry.json")
	reg, err := LoadRegistry(regPath)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}

	info, ok := reg["my-project/user"]
	if !ok {
		t.Fatal("my-project/user not in registry")
	}
	if info.Latest != "v1" {
		t.Errorf("latest = %q, want v1", info.Latest)
	}

	// Verify git commit was made
	log := runGitCmdOutput(t, cacheDir, "log", "--oneline", "-1")
	if !strings.Contains(log, "my-project/user@v1") {
		t.Errorf("commit message = %q, should contain 'my-project/user@v1'", log)
	}
}

func TestPushAPIToRegistry_NoThriftFiles(t *testing.T) {
	cacheDir := setupGitRepo(t)

	// Create initial commit
	os.WriteFile(filepath.Join(cacheDir, "README.md"), []byte("init"), 0644)
	runGitCmd(t, cacheDir, "add", "-A")
	runGitCmd(t, cacheDir, "commit", "-m", "init")

	// Source dir with no .thrift files
	sourceDir := t.TempDir()
	os.WriteFile(filepath.Join(sourceDir, "readme.md"), []byte("# Hello"), 0644)

	opts := APIPushOptions{
		CacheDir:  cacheDir,
		Project:   "my-project",
		Resource:  "user",
		Version:   "v1",
		SourceDir: sourceDir,
		DryRun:    false,
	}

	err := PushAPIToRegistry(opts)
	if err == nil {
		t.Fatal("should fail when no .thrift files in source")
	}
	if !strings.Contains(err.Error(), "no .thrift files") {
		t.Errorf("error = %q, should mention 'no .thrift files'", err.Error())
	}
}
