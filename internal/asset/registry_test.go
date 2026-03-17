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

func TestLoadRegistry_V2Format(t *testing.T) {
	dir := t.TempDir()
	regJSON := `{
		"$schema": "https://gve.dev/schema/registry.json",
		"version": "2",
		"ui/spinner": {
			"latest": "1.0.0",
			"versions": {
				"1.0.0": {"path": "ui/spinner/v1.0.0"}
			}
		},
		"scaffold/default": {
			"latest": "1.0.0",
			"versions": {
				"1.0.0": {"path": "scaffold/default/v1.0.0"}
			}
		}
	}`
	path := filepath.Join(dir, "registry.json")
	os.WriteFile(path, []byte(regJSON), 0644)

	reg, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(reg) != 2 {
		t.Errorf("registry has %d assets, want 2", len(reg))
	}
	if _, ok := reg["ui/spinner"]; !ok {
		t.Error("ui/spinner not found in registry")
	}
	if _, ok := reg["scaffold/default"]; !ok {
		t.Error("scaffold/default not found in registry")
	}
	// $schema and version should NOT be in registry
	if _, ok := reg["$schema"]; ok {
		t.Error("$schema should be filtered out")
	}
	if _, ok := reg["version"]; ok {
		t.Error("version should be filtered out")
	}
}

func TestRegistry_ResolveShortName(t *testing.T) {
	reg := Registry{
		"ui/spinner":            {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/spinner/v1.0.0"}}},
		"ui/button":             {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/button/v1.0.0"}}},
		"components/data-table": {Latest: "2.0.0", Versions: map[string]VersionEntry{"2.0.0": {Path: "components/data-table/v2.0.0"}}},
		"global/theme":          {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "global/theme/v1.0.0"}}},
	}

	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"ui/spinner", "ui/spinner", true},
		{"components/data-table", "components/data-table", true},
		{"spinner", "ui/spinner", true},              // shortname -> ui/ first
		{"data-table", "components/data-table", true}, // shortname -> components/
		{"theme", "global/theme", true},               // shortname -> global/
		{"nonexistent", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := reg.ResolveAssetName(tt.input)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ResolveAssetName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRegistry_ListByCategory(t *testing.T) {
	reg := Registry{
		"scaffold/default":      {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {}}},
		"ui/spinner":            {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {}}},
		"ui/button":             {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {}}},
		"components/data-table": {Latest: "2.0.0", Versions: map[string]VersionEntry{"2.0.0": {}}},
	}

	scaffolds := reg.ListByCategory("scaffold")
	if len(scaffolds) != 1 || scaffolds[0] != "scaffold/default" {
		t.Errorf("scaffolds = %v", scaffolds)
	}
	uis := reg.ListByCategory("ui")
	if len(uis) != 2 {
		t.Errorf("uis = %v, want 2 items", uis)
	}
	// Should be sorted
	if uis[0] != "ui/button" || uis[1] != "ui/spinner" {
		t.Errorf("uis not sorted: %v", uis)
	}
	comps := reg.ListByCategory("components")
	if len(comps) != 1 || comps[0] != "components/data-table" {
		t.Errorf("components = %v", comps)
	}
	empty := reg.ListByCategory("global")
	if len(empty) != 0 {
		t.Errorf("global = %v, want empty", empty)
	}
}
