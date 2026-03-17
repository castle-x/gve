package lock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAndSaveLoad(t *testing.T) {
	lf := New("github.com/castle-x/wk-ui", "github.com/castle-x/wk-api")
	lf.SetUIAsset("ui/button", "1.0.0")
	lf.SetAPIAsset("example/user", "v1")

	dir := t.TempDir()
	path := filepath.Join(dir, "gve.lock")

	if err := lf.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Version != "2" {
		t.Errorf("Version = %q, want %q", loaded.Version, "2")
	}
	if loaded.UI.Registry != "github.com/castle-x/wk-ui" {
		t.Errorf("UI.Registry = %q", loaded.UI.Registry)
	}

	v, ok := loaded.GetUIAsset("ui/button")
	if !ok || v != "1.0.0" {
		t.Errorf("GetUIAsset(ui/button) = %q, %v", v, ok)
	}

	v, ok = loaded.GetAPIAsset("example/user")
	if !ok || v != "v1" {
		t.Errorf("GetAPIAsset(example/user) = %q, %v", v, ok)
	}

	_, ok = loaded.GetUIAsset("nonexistent")
	if ok {
		t.Error("GetUIAsset(nonexistent) should return false")
	}
}

func TestNewV2(t *testing.T) {
	lf := New("ui-reg", "api-reg")
	if lf.Version != "2" {
		t.Errorf("Version = %q, want %q", lf.Version, "2")
	}
}

func TestLoadV1_AutoMigrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gve.lock")
	// Write a v1 lock file
	v1 := `{
		"version": "1",
		"ui": {
			"registry": "github.com/castle-x/wk-ui",
			"assets": {
				"button": {"version": "1.0.0"},
				"base-setup": {"version": "1.0.0"}
			}
		},
		"api": {
			"registry": "github.com/castle-x/wk-api",
			"assets": {}
		}
	}`
	os.WriteFile(path, []byte(v1), 0644)

	lf, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Should auto-migrate to v2
	if lf.Version != "2" {
		t.Errorf("Version = %q, want %q after migration", lf.Version, "2")
	}

	// "button" -> "ui/button"
	v, ok := lf.GetUIAsset("ui/button")
	if !ok || v != "1.0.0" {
		t.Errorf("GetUIAsset(ui/button) = %q, %v", v, ok)
	}

	// "base-setup" -> "scaffold/default"
	v, ok = lf.GetUIAsset("scaffold/default")
	if !ok || v != "1.0.0" {
		t.Errorf("GetUIAsset(scaffold/default) = %q, %v", v, ok)
	}

	// Old keys should no longer exist
	_, ok = lf.GetUIAsset("button")
	if ok {
		t.Error("old key 'button' should not exist after migration")
	}
}

func TestLoadV2_NoMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gve.lock")
	v2 := `{
		"version": "2",
		"ui": {
			"registry": "github.com/castle-x/wk-ui",
			"assets": {
				"ui/spinner": {"version": "1.0.0"},
				"scaffold/default": {"version": "1.0.0"}
			}
		},
		"api": {
			"registry": "github.com/castle-x/wk-api",
			"assets": {}
		}
	}`
	os.WriteFile(path, []byte(v2), 0644)

	lf, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if lf.Version != "2" {
		t.Errorf("Version = %q", lf.Version)
	}
	v, ok := lf.GetUIAsset("ui/spinner")
	if !ok || v != "1.0.0" {
		t.Errorf("GetUIAsset(ui/spinner) = %q, %v", v, ok)
	}
}

func TestLoadNonexistent(t *testing.T) {
	_, err := Load("/nonexistent/path/gve.lock")
	if err == nil {
		t.Error("Load nonexistent should return error")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gve.lock")
	os.WriteFile(path, []byte("{invalid}"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Error("Load invalid JSON should return error")
	}
}
