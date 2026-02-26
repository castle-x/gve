package lock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAndSaveLoad(t *testing.T) {
	lf := New("github.com/castle-x/wk-ui", "github.com/castle-x/wk-api")
	lf.SetUIAsset("button", "1.0.0")
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

	if loaded.Version != "1" {
		t.Errorf("Version = %q, want %q", loaded.Version, "1")
	}
	if loaded.UI.Registry != "github.com/castle-x/wk-ui" {
		t.Errorf("UI.Registry = %q", loaded.UI.Registry)
	}

	v, ok := loaded.GetUIAsset("button")
	if !ok || v != "1.0.0" {
		t.Errorf("GetUIAsset(button) = %q, %v", v, ok)
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
