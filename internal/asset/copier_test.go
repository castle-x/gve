package asset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyAsset(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	os.MkdirAll(filepath.Join(srcDir, "sub"), 0755)
	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(srcDir, "sub", "nested.txt"), []byte("world"), 0644)

	files := []string{"file.txt", "sub/nested.txt"}
	if err := CopyAsset(srcDir, destDir, files); err != nil {
		t.Fatalf("CopyAsset: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(destDir, "file.txt"))
	if err != nil {
		t.Fatalf("read file.txt: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("file.txt content = %q, want %q", data, "hello")
	}

	data, err = os.ReadFile(filepath.Join(destDir, "sub", "nested.txt"))
	if err != nil {
		t.Fatalf("read nested.txt: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("nested.txt content = %q, want %q", data, "world")
	}
}

func TestCopyAssetMissingSrc(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	err := CopyAsset(srcDir, destDir, []string{"nonexistent.txt"})
	if err == nil {
		t.Error("CopyAsset with missing file should return error")
	}
}
