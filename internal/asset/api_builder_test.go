package asset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildAPIRegistry_SingleResource(t *testing.T) {
	dir := t.TempDir()

	// Create project/resource/v1 with a .thrift file
	vDir := filepath.Join(dir, "my-project", "user", "v1")
	os.MkdirAll(vDir, 0755)
	os.WriteFile(filepath.Join(vDir, "user.thrift"), []byte("service User {}"), 0644)

	reg, err := BuildAPIRegistry(dir)
	if err != nil {
		t.Fatalf("BuildAPIRegistry: %v", err)
	}

	info, ok := reg["my-project/user"]
	if !ok {
		t.Fatal("my-project/user not in registry")
	}
	if info.Latest != "v1" {
		t.Errorf("latest = %q, want v1", info.Latest)
	}
	if len(info.Versions) != 1 {
		t.Errorf("version count = %d, want 1", len(info.Versions))
	}
	ve, ok := info.Versions["v1"]
	if !ok {
		t.Fatal("v1 not in versions")
	}
	if ve.Path != filepath.Join("my-project", "user", "v1") {
		t.Errorf("path = %q, want my-project/user/v1", ve.Path)
	}
}

func TestBuildAPIRegistry_MultiVersion(t *testing.T) {
	dir := t.TempDir()

	// v1 and v2
	for _, v := range []string{"v1", "v2"} {
		vDir := filepath.Join(dir, "ai-worker", "task", v)
		os.MkdirAll(vDir, 0755)
		os.WriteFile(filepath.Join(vDir, "task.thrift"), []byte("service Task {}"), 0644)
	}

	reg, err := BuildAPIRegistry(dir)
	if err != nil {
		t.Fatalf("BuildAPIRegistry: %v", err)
	}

	info, ok := reg["ai-worker/task"]
	if !ok {
		t.Fatal("ai-worker/task not in registry")
	}
	if info.Latest != "v2" {
		t.Errorf("latest = %q, want v2", info.Latest)
	}
	if len(info.Versions) != 2 {
		t.Errorf("version count = %d, want 2", len(info.Versions))
	}
}

func TestBuildAPIRegistry_NoThriftFiles(t *testing.T) {
	dir := t.TempDir()

	// Directory exists but no .thrift files
	vDir := filepath.Join(dir, "my-project", "empty", "v1")
	os.MkdirAll(vDir, 0755)
	os.WriteFile(filepath.Join(vDir, "readme.txt"), []byte("no thrift here"), 0644)

	reg, err := BuildAPIRegistry(dir)
	if err != nil {
		t.Fatalf("BuildAPIRegistry: %v", err)
	}

	if _, ok := reg["my-project/empty"]; ok {
		t.Error("should not include resource with no thrift files")
	}
}

func TestBuildAPIRegistry_NonVersionDir(t *testing.T) {
	dir := t.TempDir()

	// Create non-version directories alongside a valid one
	vDir := filepath.Join(dir, "my-project", "user", "v1")
	os.MkdirAll(vDir, 0755)
	os.WriteFile(filepath.Join(vDir, "user.thrift"), []byte("service User {}"), 0644)

	// Non-version dirs should be skipped
	os.MkdirAll(filepath.Join(dir, "my-project", "user", "docs"), 0755)
	os.MkdirAll(filepath.Join(dir, "my-project", "user", "latest"), 0755)

	reg, err := BuildAPIRegistry(dir)
	if err != nil {
		t.Fatalf("BuildAPIRegistry: %v", err)
	}

	info := reg["my-project/user"]
	if len(info.Versions) != 1 {
		t.Errorf("version count = %d, want 1 (should skip non-vN dirs)", len(info.Versions))
	}
}

func TestBuildAPIRegistry_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()

	// .git should be skipped
	os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0755)

	// Valid entry
	vDir := filepath.Join(dir, "my-project", "user", "v1")
	os.MkdirAll(vDir, 0755)
	os.WriteFile(filepath.Join(vDir, "user.thrift"), []byte("service User {}"), 0644)

	reg, err := BuildAPIRegistry(dir)
	if err != nil {
		t.Fatalf("BuildAPIRegistry: %v", err)
	}

	if _, ok := reg[".git/objects"]; ok {
		t.Error("should not include .git directory")
	}
	if _, ok := reg["my-project/user"]; !ok {
		t.Error("should include valid entry")
	}
}

func TestWriteAPIRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	reg := Registry{
		"my-project/user": {
			Latest: "v1",
			Versions: map[string]VersionEntry{
				"v1": {Path: "my-project/user/v1"},
			},
		},
	}

	if err := WriteAPIRegistry(reg, path); err != nil {
		t.Fatalf("WriteAPIRegistry: %v", err)
	}

	// Reload and verify
	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}

	info, ok := loaded["my-project/user"]
	if !ok {
		t.Fatal("my-project/user not in loaded registry")
	}
	if info.Latest != "v1" {
		t.Errorf("latest = %q, want v1", info.Latest)
	}
}
