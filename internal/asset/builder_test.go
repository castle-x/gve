package asset

import (
	"encoding/json"
	"os"
	"path/filepath"
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
