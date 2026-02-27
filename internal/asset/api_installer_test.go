package asset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallAPIAsset(t *testing.T) {
	cacheDir := t.TempDir()
	projectDir := t.TempDir()

	apiDir := filepath.Join(cacheDir, "api", "my-project", "user", "v1")
	os.MkdirAll(apiDir, 0755)
	os.WriteFile(filepath.Join(apiDir, "user.thrift"), []byte("service User {}"), 0644)
	os.WriteFile(filepath.Join(apiDir, "user.go"), []byte("package user"), 0644)
	os.WriteFile(filepath.Join(apiDir, "client.go"), []byte("package user // client"), 0644)
	os.WriteFile(filepath.Join(apiDir, "client.ts"), []byte("export class UserClient {}"), 0644)

	reg := Registry{
		"my-project/user": {
			Latest: "v1",
			Versions: map[string]VersionEntry{
				"v1": {Path: "my-project/user/v1"},
			},
		},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(cacheDir, "api", "registry.json"), regData, 0644)

	mgr := NewManager(cacheDir)

	ver, err := InstallAPIAsset(mgr, "my-project/user", "v1", projectDir)
	if err != nil {
		t.Fatalf("InstallAPIAsset: %v", err)
	}
	if ver != "v1" {
		t.Errorf("version = %q, want %q", ver, "v1")
	}

	expectedFiles := []string{"user.thrift", "user.go", "client.go", "client.ts"}
	destDir := filepath.Join(projectDir, "api", "my-project", "user", "v1")
	for _, f := range expectedFiles {
		if _, err := os.Stat(filepath.Join(destDir, f)); err != nil {
			t.Errorf("expected file %s not found: %v", f, err)
		}
	}
}

func TestInstallAPIAsset_Latest(t *testing.T) {
	cacheDir := t.TempDir()
	projectDir := t.TempDir()

	apiDir := filepath.Join(cacheDir, "api", "my-project", "task", "v2")
	os.MkdirAll(apiDir, 0755)
	os.WriteFile(filepath.Join(apiDir, "task.thrift"), []byte("service Task {}"), 0644)

	reg := Registry{
		"my-project/task": {
			Latest: "v2",
			Versions: map[string]VersionEntry{
				"v1": {Path: "my-project/task/v1"},
				"v2": {Path: "my-project/task/v2"},
			},
		},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(cacheDir, "api", "registry.json"), regData, 0644)

	mgr := NewManager(cacheDir)

	ver, err := InstallAPIAsset(mgr, "my-project/task", "", projectDir)
	if err != nil {
		t.Fatalf("InstallAPIAsset latest: %v", err)
	}
	if ver != "v2" {
		t.Errorf("version = %q, want %q", ver, "v2")
	}
}

// TestAPIAssetDirExists_AfterManualDelete simulates the QA-reported bug:
// when the api directory is manually deleted, APIAssetDirExists returns false
// so api sync can detect the missing files and reinstall.
func TestAPIAssetDirExists_AfterManualDelete(t *testing.T) {
	projectDir := t.TempDir()
	dir := filepath.Join(projectDir, "api", "my-project", "user", "v1")

	os.MkdirAll(dir, 0755)
	if !APIAssetDirExists(projectDir, "my-project/user", "v1") {
		t.Fatal("expected true after mkdir")
	}

	os.RemoveAll(dir)
	if APIAssetDirExists(projectDir, "my-project/user", "v1") {
		t.Error("expected false after manual deletion — api sync must reinstall")
	}
}

func TestAPIAssetDirExists(t *testing.T) {
	projectDir := t.TempDir()

	if APIAssetDirExists(projectDir, "my-project/user", "v1") {
		t.Error("expected false for non-existent dir")
	}

	os.MkdirAll(filepath.Join(projectDir, "api", "my-project", "user", "v1"), 0755)

	if !APIAssetDirExists(projectDir, "my-project/user", "v1") {
		t.Error("expected true for existing dir")
	}
}
