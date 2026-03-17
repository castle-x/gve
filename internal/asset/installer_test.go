package asset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallUIAsset(t *testing.T) {
	// Create temp cache with registry + one asset
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	projectDir := filepath.Join(tmp, "project")

	// registry.json
	regDir := filepath.Join(cacheDir, "ui")
	os.MkdirAll(regDir, 0755)
	reg := Registry{
		"button": AssetInfo{
			Latest: "1.0.0",
			Versions: map[string]VersionEntry{
				"1.0.0": {Path: "assets/button/v1.0.0"},
			},
		},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(regDir, "registry.json"), regData, 0644)

	// assets/button/v1.0.0/
	assetDir := filepath.Join(regDir, "assets", "button", "v1.0.0")
	os.MkdirAll(assetDir, 0755)
	meta := Meta{Name: "button", Version: "1.0.0", Deps: []string{"@radix-ui/react-slot"}, Files: []string{"button.tsx"}}
	metaData, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(assetDir, "meta.json"), metaData, 0644)
	os.WriteFile(filepath.Join(assetDir, "button.tsx"), []byte("export const Button = () => null"), 0644)

	// project/site/package.json
	siteDir := filepath.Join(projectDir, "site")
	os.MkdirAll(siteDir, 0755)
	pkg := map[string]interface{}{
		"name": "site",
		"dependencies": map[string]interface{}{
			"react": "^19.0.0",
		},
	}
	pkgData, _ := json.MarshalIndent(pkg, "", "  ")
	os.WriteFile(filepath.Join(siteDir, "package.json"), pkgData, 0644)

	mgr := NewManager(cacheDir)

	ver, err := InstallUIAsset(mgr, "button", "1.0.0", projectDir)
	if err != nil {
		t.Fatalf("InstallUIAsset: %v", err)
	}
	if ver != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", ver)
	}

	// v1 assets with no category infer "" from "assets/..." path, fallback to shared/wk/ui/
	destFile := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "button", "button.tsx")
	body, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}
	if string(body) != "export const Button = () => null" {
		t.Errorf("copied content mismatch: %s", body)
	}

	// Check deps injected
	pkgPath := filepath.Join(siteDir, "package.json")
	data, _ := os.ReadFile(pkgPath)
	var outPkg map[string]interface{}
	json.Unmarshal(data, &outPkg)
	deps, _ := outPkg["dependencies"].(map[string]interface{})
	if deps["@radix-ui/react-slot"] == nil {
		t.Errorf("package.json missing @radix-ui/react-slot, got %v", deps)
	}
}

func TestInstallUIAsset_WithDest(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	projectDir := filepath.Join(tmp, "project")

	regDir := filepath.Join(cacheDir, "ui")
	os.MkdirAll(regDir, 0755)
	reg := Registry{
		"theme": AssetInfo{
			Latest: "1.0.0",
			Versions: map[string]VersionEntry{
				"1.0.0": {Path: "assets/theme/v1.0.0"},
			},
		},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(regDir, "registry.json"), regData, 0644)

	assetDir := filepath.Join(regDir, "assets", "theme", "v1.0.0")
	os.MkdirAll(assetDir, 0755)
	meta := Meta{Name: "theme", Version: "1.0.0", Dest: "site/src/app/styles", Files: []string{"globals.css"}}
	metaData, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(assetDir, "meta.json"), metaData, 0644)
	os.WriteFile(filepath.Join(assetDir, "globals.css"), []byte(":root {}"), 0644)

	mgr := NewManager(cacheDir)

	ver, err := InstallUIAsset(mgr, "theme", "latest", projectDir)
	if err != nil {
		t.Fatalf("InstallUIAsset: %v", err)
	}
	if ver != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", ver)
	}

	destFile := filepath.Join(projectDir, "site", "src", "app", "styles", "globals.css")
	if _, err := os.Stat(destFile); err != nil {
		t.Fatalf("expected file at dest: %v", err)
	}
}

func TestInstallUIAsset_V2_UICategory(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	projectDir := filepath.Join(tmp, "project")

	regDir := filepath.Join(cacheDir, "ui")
	os.MkdirAll(regDir, 0755)
	reg := Registry{
		"ui/spinner": AssetInfo{
			Latest: "1.0.0",
			Versions: map[string]VersionEntry{
				"1.0.0": {Path: "ui/spinner/v1.0.0"},
			},
		},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(regDir, "registry.json"), regData, 0644)

	assetDir := filepath.Join(regDir, "ui", "spinner", "v1.0.0")
	os.MkdirAll(assetDir, 0755)
	meta := Meta{Name: "spinner", Version: "1.0.0", Category: "ui", Files: []string{"spinner.tsx"}}
	metaData, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(assetDir, "meta.json"), metaData, 0644)
	os.WriteFile(filepath.Join(assetDir, "spinner.tsx"), []byte("export const Spinner = () => null"), 0644)

	siteDir := filepath.Join(projectDir, "site")
	os.MkdirAll(siteDir, 0755)
	os.WriteFile(filepath.Join(siteDir, "package.json"), []byte(`{"name":"app","dependencies":{}}`), 0644)

	mgr := NewManager(cacheDir)

	ver, err := InstallUIAsset(mgr, "ui/spinner", "1.0.0", projectDir)
	if err != nil {
		t.Fatalf("InstallUIAsset: %v", err)
	}
	if ver != "1.0.0" {
		t.Errorf("version = %q", ver)
	}

	// Should install to shared/wk/ui/spinner/
	destFile := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "spinner", "spinner.tsx")
	if _, err := os.Stat(destFile); err != nil {
		t.Fatalf("spinner.tsx not installed to v2 path: %v", err)
	}
}

func TestInstallUIAsset_V2_ComponentCategory(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	projectDir := filepath.Join(tmp, "project")

	regDir := filepath.Join(cacheDir, "ui")
	os.MkdirAll(regDir, 0755)
	reg := Registry{
		"components/data-table": AssetInfo{
			Latest: "2.0.0",
			Versions: map[string]VersionEntry{
				"2.0.0": {Path: "components/data-table/v2.0.0"},
			},
		},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(regDir, "registry.json"), regData, 0644)

	assetDir := filepath.Join(regDir, "components", "data-table", "v2.0.0")
	os.MkdirAll(assetDir, 0755)
	meta := Meta{Name: "data-table", Version: "2.0.0", Category: "component", Files: []string{"data-table.tsx"}}
	metaData, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(assetDir, "meta.json"), metaData, 0644)
	os.WriteFile(filepath.Join(assetDir, "data-table.tsx"), []byte("export const DataTable = () => null"), 0644)

	siteDir := filepath.Join(projectDir, "site")
	os.MkdirAll(siteDir, 0755)
	os.WriteFile(filepath.Join(siteDir, "package.json"), []byte(`{"name":"app","dependencies":{}}`), 0644)

	mgr := NewManager(cacheDir)

	ver, err := InstallUIAsset(mgr, "components/data-table", "2.0.0", projectDir)
	if err != nil {
		t.Fatalf("InstallUIAsset: %v", err)
	}
	if ver != "2.0.0" {
		t.Errorf("version = %q", ver)
	}

	// Should install to shared/wk/components/data-table/
	destFile := filepath.Join(projectDir, "site", "src", "shared", "wk", "components", "data-table", "data-table.tsx")
	if _, err := os.Stat(destFile); err != nil {
		t.Fatalf("data-table.tsx not installed to v2 path: %v", err)
	}
}

func TestGetInstallPath(t *testing.T) {
	tests := []struct {
		category string
		name     string
		dest     string
		want     string
	}{
		{"ui", "spinner", "", "site/src/shared/wk/ui/spinner"},
		{"component", "data-table", "", "site/src/shared/wk/components/data-table"},
		{"global", "theme", "site/src/app/styles", "site/src/app/styles"},
		{"scaffold", "default", "site", "site"},
		{"scaffold", "default", "", "site"},
		{"", "button", "", "site/src/shared/wk/ui/button"}, // fallback
	}
	for _, tt := range tests {
		t.Run(tt.category+"/"+tt.name, func(t *testing.T) {
			got := GetInstallPath(tt.category, tt.name, tt.dest)
			if got != tt.want {
				t.Errorf("GetInstallPath(%q,%q,%q) = %q, want %q", tt.category, tt.name, tt.dest, got, tt.want)
			}
		})
	}
}
