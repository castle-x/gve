package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckPort_Available(t *testing.T) {
	// Port 0 lets the OS pick a free port — we use a high random port
	// This test simply verifies checkPort doesn't error on a likely-free port
	err := checkPort(0)
	if err != nil {
		t.Errorf("port 0 should be available: %v", err)
	}
}

func TestReadPIDFile_Missing(t *testing.T) {
	_, running := readPIDFile("/tmp/nonexistent-gve-pid-file")
	if running {
		t.Error("expected running=false for missing file")
	}
}

func TestReadPIDFile_InvalidContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.pid")
	os.WriteFile(path, []byte("not-a-number"), 0644)

	_, running := readPIDFile(path)
	if running {
		t.Error("expected running=false for invalid PID content")
	}
}

func TestReadPIDFile_DeadProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.pid")
	// PID 999999 is almost certainly not running
	os.WriteFile(path, []byte("999999"), 0644)

	_, running := readPIDFile(path)
	if running {
		t.Error("expected running=false for dead process")
	}
	// PID file should be cleaned up
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("stale PID file should have been removed")
	}
}

func TestNeedsRebuild_NoBinary(t *testing.T) {
	dir := t.TempDir()
	needs, err := needsRebuild(dir, filepath.Join(dir, "dist", "myapp"))
	if err != nil {
		t.Fatal(err)
	}
	if !needs {
		t.Error("should need rebuild when binary doesn't exist")
	}
}

func TestNeedsRebuild_BinaryNewer(t *testing.T) {
	dir := t.TempDir()

	// Create a source file
	srcDir := filepath.Join(dir, "cmd", "server")
	os.MkdirAll(srcDir, 0755)
	srcFile := filepath.Join(srcDir, "main.go")
	os.WriteFile(srcFile, []byte("package main"), 0644)

	// Create binary with future timestamp
	distDir := filepath.Join(dir, "dist")
	os.MkdirAll(distDir, 0755)
	binPath := filepath.Join(distDir, "myapp")
	os.WriteFile(binPath, []byte("binary"), 0755)
	future := time.Now().Add(1 * time.Hour)
	os.Chtimes(binPath, future, future)

	needs, err := needsRebuild(dir, binPath)
	if err != nil {
		t.Fatal(err)
	}
	if needs {
		t.Error("should not need rebuild when binary is newer than sources")
	}
}

func TestNeedsRebuild_SourceNewer(t *testing.T) {
	dir := t.TempDir()

	// Create binary with old timestamp
	distDir := filepath.Join(dir, "dist")
	os.MkdirAll(distDir, 0755)
	binPath := filepath.Join(distDir, "myapp")
	os.WriteFile(binPath, []byte("binary"), 0755)
	past := time.Now().Add(-1 * time.Hour)
	os.Chtimes(binPath, past, past)

	// Create a newer source file
	srcDir := filepath.Join(dir, "cmd", "server")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644)

	needs, err := needsRebuild(dir, binPath)
	if err != nil {
		t.Fatal(err)
	}
	if !needs {
		t.Error("should need rebuild when source is newer than binary")
	}
}

func TestNewerThan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("x"), 0644)

	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	if !newerThan(path, past) {
		t.Error("file should be newer than past reference")
	}
	if newerThan(path, future) {
		t.Error("file should not be newer than future reference")
	}
	if newerThan(filepath.Join(dir, "nonexistent"), past) {
		t.Error("nonexistent file should not be newer")
	}
}

func TestDirHasNewerFile(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "file.go"), []byte("x"), 0644)

	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	newer, _ := dirHasNewerFile(dir, past)
	if !newer {
		t.Error("dir should have newer file than past ref")
	}

	newer, _ = dirHasNewerFile(dir, future)
	if newer {
		t.Error("dir should not have newer file than future ref")
	}
}

func TestDirHasNewerFile_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	nmDir := filepath.Join(dir, "node_modules", "pkg")
	os.MkdirAll(nmDir, 0755)
	os.WriteFile(filepath.Join(nmDir, "index.js"), []byte("x"), 0644)

	past := time.Now().Add(-1 * time.Hour)
	newer, _ := dirHasNewerFile(dir, past)
	if newer {
		t.Error("should skip node_modules directory")
	}
}
