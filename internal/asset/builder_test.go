package asset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRegistry(t *testing.T) {
	dir := t.TempDir()
	assetsDir := filepath.Join(dir, "assets")

	// Create button v1.0.0
	btnDir := filepath.Join(assetsDir, "button", "v1.0.0")
	os.MkdirAll(btnDir, 0755)
	meta := Meta{Name: "button", Version: "1.0.0", Files: []string{"button.tsx"}}
	data, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(btnDir, "meta.json"), data, 0644)
	os.WriteFile(filepath.Join(btnDir, "button.tsx"), []byte("export const Button = () => {}"), 0644)

	// Create button v1.1.0
	btn2Dir := filepath.Join(assetsDir, "button", "v1.1.0")
	os.MkdirAll(btn2Dir, 0755)
	meta2 := Meta{Name: "button", Version: "1.1.0", Files: []string{"button.tsx"}}
	data2, _ := json.Marshal(meta2)
	os.WriteFile(filepath.Join(btn2Dir, "meta.json"), data2, 0644)
	os.WriteFile(filepath.Join(btn2Dir, "button.tsx"), []byte("export const Button = () => {}"), 0644)

	// Create theme v1.0.0
	themeDir := filepath.Join(assetsDir, "theme", "v1.0.0")
	os.MkdirAll(themeDir, 0755)
	meta3 := Meta{Name: "theme", Version: "1.0.0", Dest: "site/src/app/styles", Files: []string{"globals.css"}}
	data3, _ := json.Marshal(meta3)
	os.WriteFile(filepath.Join(themeDir, "meta.json"), data3, 0644)
	os.WriteFile(filepath.Join(themeDir, "globals.css"), []byte(":root {}"), 0644)

	reg, err := BuildRegistry(assetsDir)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}

	if len(reg) != 2 {
		t.Errorf("expected 2 assets, got %d", len(reg))
	}

	btnInfo, ok := reg["button"]
	if !ok {
		t.Fatal("button not found in registry")
	}
	if btnInfo.Latest != "1.1.0" {
		t.Errorf("button latest = %q, want %q", btnInfo.Latest, "1.1.0")
	}
	if len(btnInfo.Versions) != 2 {
		t.Errorf("button versions = %d, want 2", len(btnInfo.Versions))
	}

	themeInfo, ok := reg["theme"]
	if !ok {
		t.Fatal("theme not found in registry")
	}
	if themeInfo.Latest != "1.0.0" {
		t.Errorf("theme latest = %q, want %q", themeInfo.Latest, "1.0.0")
	}
}

func TestBuildRegistryEmpty(t *testing.T) {
	dir := t.TempDir()
	reg, err := BuildRegistry(dir)
	if err != nil {
		t.Fatalf("BuildRegistry empty: %v", err)
	}
	if len(reg) != 0 {
		t.Errorf("expected empty registry, got %d assets", len(reg))
	}
}

// Test helpers for v2 tests

func writeMeta(t *testing.T, dir, name, version, category string) {
	t.Helper()
	m := Meta{Name: name, Version: version, Category: category, Files: []string{name + ".tsx"}}
	data, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
}

func writeMetaWithPeerDeps(t *testing.T, dir, name, version, category string, peerDeps []string) {
	t.Helper()
	m := Meta{Name: name, Version: version, Category: category, PeerDeps: peerDeps, Files: []string{name + ".tsx"}}
	data, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
}


func TestBuildRegistryV2(t *testing.T) {
	dir := t.TempDir()

	// scaffold/default/v1.0.0
	d := filepath.Join(dir, "scaffold", "default", "v1.0.0")
	os.MkdirAll(d, 0755)
	writeMeta(t, d, "default", "1.0.0", "scaffold")
	os.WriteFile(filepath.Join(d, "embed.go"), []byte("package site"), 0644)

	// ui/spinner/v1.0.0
	d = filepath.Join(dir, "ui", "spinner", "v1.0.0")
	os.MkdirAll(d, 0755)
	writeMeta(t, d, "spinner", "1.0.0", "ui")
	os.WriteFile(filepath.Join(d, "spinner.tsx"), []byte("export const Spinner = () => null"), 0644)

	// components/data-table/v2.0.0
	d = filepath.Join(dir, "components", "data-table", "v2.0.0")
	os.MkdirAll(d, 0755)
	writeMetaWithPeerDeps(t, d, "data-table", "2.0.0", "component", []string{"ui/spinner"})
	os.WriteFile(filepath.Join(d, "data-table.tsx"), []byte("export const DataTable = () => null"), 0644)

	reg, warnings, err := BuildRegistryV2(dir)
	if err != nil {
		t.Fatalf("BuildRegistryV2: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	// Check keys
	expected := []string{"scaffold/default", "ui/spinner", "components/data-table"}
	for _, key := range expected {
		if _, ok := reg[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}

	// Check paths
	if reg["ui/spinner"].Versions["1.0.0"].Path != "ui/spinner/v1.0.0" {
		t.Errorf("spinner path = %q", reg["ui/spinner"].Versions["1.0.0"].Path)
	}
	if reg["scaffold/default"].Versions["1.0.0"].Path != "scaffold/default/v1.0.0" {
		t.Errorf("scaffold path = %q", reg["scaffold/default"].Versions["1.0.0"].Path)
	}
	if reg["components/data-table"].Versions["2.0.0"].Path != "components/data-table/v2.0.0" {
		t.Errorf("data-table path = %q", reg["components/data-table"].Versions["2.0.0"].Path)
	}
}

func TestBuildRegistryV2_CSSWarning(t *testing.T) {
	dir := t.TempDir()

	// ui/bad-component with .css file in meta.json files list
	d := filepath.Join(dir, "ui", "bad", "v1.0.0")
	os.MkdirAll(d, 0755)
	meta := Meta{Name: "bad", Version: "1.0.0", Category: "ui", Files: []string{"bad.tsx", "bad.module.css"}}
	data, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(d, "meta.json"), data, 0644)
	os.WriteFile(filepath.Join(d, "bad.tsx"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(d, "bad.module.css"), []byte(".x{}"), 0644)

	_, warnings, err := BuildRegistryV2(dir)
	if err != nil {
		t.Fatalf("BuildRegistryV2: %v", err)
	}
	if len(warnings) == 0 {
		t.Error("expected CSS warning for ui/ asset with .css file")
	}
	foundCSS := false
	for _, w := range warnings {
		if strings.Contains(w, "CSS") {
			foundCSS = true
		}
	}
	if !foundCSS {
		t.Errorf("expected CSS-related warning, got: %v", warnings)
	}
}

func TestBuildRegistryV2_CategoryMismatchWarning(t *testing.T) {
	dir := t.TempDir()

	// ui/misplaced with category "component" in meta.json
	d := filepath.Join(dir, "ui", "misplaced", "v1.0.0")
	os.MkdirAll(d, 0755)
	meta := Meta{Name: "misplaced", Version: "1.0.0", Category: "component", Files: []string{"misplaced.tsx"}}
	data, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(d, "meta.json"), data, 0644)
	os.WriteFile(filepath.Join(d, "misplaced.tsx"), []byte("x"), 0644)

	_, warnings, err := BuildRegistryV2(dir)
	if err != nil {
		t.Fatalf("BuildRegistryV2: %v", err)
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "mismatch") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected category mismatch warning, got: %v", warnings)
	}
}

func TestBuildRegistryV2_UndeclaredCSSWarning(t *testing.T) {
	dir := t.TempDir()

	// ui/sneaky has a .css file on disk but NOT declared in meta.json
	d := filepath.Join(dir, "ui", "sneaky", "v1.0.0")
	os.MkdirAll(d, 0755)
	meta := Meta{Name: "sneaky", Version: "1.0.0", Category: "ui", Files: []string{"sneaky.tsx"}}
	data, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(d, "meta.json"), data, 0644)
	os.WriteFile(filepath.Join(d, "sneaky.tsx"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(d, "hidden.css"), []byte(".x{}"), 0644) // undeclared!

	_, warnings, err := BuildRegistryV2(dir)
	if err != nil {
		t.Fatalf("BuildRegistryV2: %v", err)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "undeclared CSS") && strings.Contains(w, "hidden.css") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected undeclared CSS warning for hidden.css, got: %v", warnings)
	}
}

func TestBuildRegistryV2_EmptyDirs(t *testing.T) {
	dir := t.TempDir()
	// No category dirs at all
	reg, warnings, err := BuildRegistryV2(dir)
	if err != nil {
		t.Fatalf("BuildRegistryV2 empty: %v", err)
	}
	if len(reg) != 0 {
		t.Errorf("expected empty registry, got %d assets", len(reg))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got: %v", warnings)
	}
}

func TestBuildRegistryV2Filtered(t *testing.T) {
	dir := t.TempDir()

	// ui/spinner/v1.0.0
	d := filepath.Join(dir, "ui", "spinner", "v1.0.0")
	os.MkdirAll(d, 0755)
	writeMeta(t, d, "spinner", "1.0.0", "ui")
	os.WriteFile(filepath.Join(d, "spinner.tsx"), []byte("export const Spinner = () => null"), 0644)

	// components/data-table/v2.0.0
	d = filepath.Join(dir, "components", "data-table", "v2.0.0")
	os.MkdirAll(d, 0755)
	writeMeta(t, d, "data-table", "2.0.0", "component")
	os.WriteFile(filepath.Join(d, "data-table.tsx"), []byte("export const DataTable = () => null"), 0644)

	// Build only components
	reg, warnings, err := BuildRegistryV2Filtered(dir, []string{"components"})
	if err != nil {
		t.Fatalf("BuildRegistryV2Filtered: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	// Should have only components
	if _, ok := reg["components/data-table"]; !ok {
		t.Error("expected components/data-table in filtered registry")
	}
	if _, ok := reg["ui/spinner"]; ok {
		t.Error("ui/spinner should not be in filtered registry (only components selected)")
	}
}

func TestBuildRegistryV2Filtered_MergeWithExisting(t *testing.T) {
	dir := t.TempDir()

	// Write an existing registry.json with ui/spinner
	existingReg := Registry{
		"ui/spinner": {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/spinner/v1.0.0"}}},
	}
	existingPath := filepath.Join(dir, "registry.json")
	if err := WriteRegistryV2(existingReg, existingPath); err != nil {
		t.Fatalf("WriteRegistryV2 existing: %v", err)
	}

	// Create components/data-table/v2.0.0 on disk
	d := filepath.Join(dir, "components", "data-table", "v2.0.0")
	os.MkdirAll(d, 0755)
	writeMeta(t, d, "data-table", "2.0.0", "component")
	os.WriteFile(filepath.Join(d, "data-table.tsx"), []byte("export const DataTable = () => null"), 0644)

	// Build only components — ui entries should be preserved from existing
	reg, _, err := BuildRegistryV2Filtered(dir, []string{"components"})
	if err != nil {
		t.Fatalf("BuildRegistryV2Filtered: %v", err)
	}

	// ui/spinner should be preserved from existing registry
	if _, ok := reg["ui/spinner"]; !ok {
		t.Error("ui/spinner should be preserved from existing registry")
	}

	// components/data-table should be freshly built
	if _, ok := reg["components/data-table"]; !ok {
		t.Error("components/data-table should be in rebuilt registry")
	}

	if len(reg) != 2 {
		t.Errorf("expected 2 entries, got %d", len(reg))
	}
}

func TestWriteRegistryV2(t *testing.T) {
	dir := t.TempDir()
	reg := Registry{
		"ui/spinner": {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/spinner/v1.0.0"}}},
	}

	path := filepath.Join(dir, "registry.json")
	err := WriteRegistryV2(reg, path)
	if err != nil {
		t.Fatalf("WriteRegistryV2: %v", err)
	}

	// Verify it can be loaded back
	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry after WriteRegistryV2: %v", err)
	}
	if _, ok := loaded["ui/spinner"]; !ok {
		t.Error("ui/spinner not found after roundtrip")
	}
	// $schema and version should be filtered by LoadRegistry
	if len(loaded) != 1 {
		t.Errorf("loaded %d entries, want 1", len(loaded))
	}
}
