package asset

import (
	"os"
	"path/filepath"
	"testing"
)

const testRegistryJSON = `{
  "button": {
    "latest": "1.2.0",
    "versions": {
      "1.0.0": { "path": "assets/button/v1.0.0" },
      "1.1.0": { "path": "assets/button/v1.1.0" },
      "1.2.0": { "path": "assets/button/v1.2.0" }
    }
  },
  "base-setup": {
    "latest": "1.0.0",
    "versions": {
      "1.0.0": { "path": "assets/base-setup/v1.0.0" }
    }
  }
}`

func writeTestRegistry(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(path, []byte(testRegistryJSON), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadRegistry(t *testing.T) {
	path := writeTestRegistry(t)
	reg, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(reg) != 2 {
		t.Errorf("registry has %d assets, want 2", len(reg))
	}
}

func TestGetLatest(t *testing.T) {
	path := writeTestRegistry(t)
	reg, _ := LoadRegistry(path)

	ver, p, err := reg.GetLatest("button")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if ver != "1.2.0" {
		t.Errorf("version = %q, want %q", ver, "1.2.0")
	}
	if p != "assets/button/v1.2.0" {
		t.Errorf("path = %q, want %q", p, "assets/button/v1.2.0")
	}

	_, _, err = reg.GetLatest("nonexistent")
	if err == nil {
		t.Error("GetLatest(nonexistent) should return error")
	}
}

func TestGetVersion(t *testing.T) {
	path := writeTestRegistry(t)
	reg, _ := LoadRegistry(path)

	p, err := reg.GetVersion("button", "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if p != "assets/button/v1.0.0" {
		t.Errorf("path = %q, want %q", p, "assets/button/v1.0.0")
	}

	p, err = reg.GetVersion("button", "^1.0.0")
	if err != nil {
		t.Fatalf("GetVersion caret: %v", err)
	}
	if p != "assets/button/v1.2.0" {
		t.Errorf("caret path = %q, want %q", p, "assets/button/v1.2.0")
	}

	p, err = reg.GetVersion("button", "latest")
	if err != nil {
		t.Fatalf("GetVersion latest: %v", err)
	}
	if p != "assets/button/v1.2.0" {
		t.Errorf("latest path = %q, want %q", p, "assets/button/v1.2.0")
	}
}

func TestListAssets(t *testing.T) {
	path := writeTestRegistry(t)
	reg, _ := LoadRegistry(path)

	names := reg.ListAssets()
	if len(names) != 2 {
		t.Errorf("ListAssets length = %d, want 2", len(names))
	}
}
