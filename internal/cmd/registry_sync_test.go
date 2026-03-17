package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/castle-x/gve/internal/asset"
)

func TestRunRegistryBuild(t *testing.T) {
	dir := t.TempDir()

	// Create v2 assets layout: ui/button/v1.0.0/
	btnDir := filepath.Join(dir, "ui", "button", "v1.0.0")
	os.MkdirAll(btnDir, 0755)
	meta := `{"name":"button","version":"1.0.0","category":"ui","files":["button.tsx"]}`
	os.WriteFile(filepath.Join(btnDir, "meta.json"), []byte(meta), 0644)
	os.WriteFile(filepath.Join(btnDir, "button.tsx"), []byte(""), 0644)

	err := runRegistryBuild(dir)
	if err != nil {
		t.Fatalf("runRegistryBuild: %v", err)
	}

	regPath := filepath.Join(dir, "registry.json")
	data, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatalf("read registry.json: %v", err)
	}

	reg, err := asset.LoadRegistry(regPath)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if _, ok := reg["ui/button"]; !ok {
		t.Errorf("registry should contain ui/button, got %s", data)
	}
	if reg["ui/button"].Latest != "1.0.0" {
		t.Errorf("button latest = %q, want 1.0.0", reg["ui/button"].Latest)
	}
}

func TestAssetExists(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	cacheDir := filepath.Join(tmp, "cache", "ui")

	// Registry + meta for ui/button (v2 format)
	reg := asset.Registry{
		"ui/button": asset.AssetInfo{
			Latest: "1.0.0",
			Versions: map[string]asset.VersionEntry{
				"1.0.0": {Path: "ui/button/v1.0.0"},
			},
		},
	}
	assetDir := filepath.Join(cacheDir, "ui", "button", "v1.0.0")
	os.MkdirAll(assetDir, 0755)
	meta := `{"name":"button","version":"1.0.0","category":"ui","files":["button.tsx"]}`
	os.WriteFile(filepath.Join(assetDir, "meta.json"), []byte(meta), 0644)

	mgr := asset.NewManager(filepath.Join(tmp, "cache"))

	// Asset not present
	if assetExists("ui/button", "1.0.0", reg, mgr, projectDir) {
		t.Error("assetExists should be false when file missing")
	}

	// Create dest file at v2 path
	destDir := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "button")
	os.MkdirAll(destDir, 0755)
	os.WriteFile(filepath.Join(destDir, "button.tsx"), nil, 0644)

	if !assetExists("ui/button", "1.0.0", reg, mgr, projectDir) {
		t.Error("assetExists should be true when first file present")
	}
}
