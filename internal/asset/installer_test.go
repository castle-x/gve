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

func TestResolvePeerDeps(t *testing.T) {
	tests := []struct {
		name      string
		peerDeps  []string
		installed map[string]bool
		want      []string
	}{
		{
			name:      "all missing",
			peerDeps:  []string{"ui/button", "ui/spinner"},
			installed: map[string]bool{},
			want:      []string{"ui/button", "ui/spinner"},
		},
		{
			name:      "some installed",
			peerDeps:  []string{"ui/button", "ui/spinner"},
			installed: map[string]bool{"ui/button": true},
			want:      []string{"ui/spinner"},
		},
		{
			name:      "all installed",
			peerDeps:  []string{"ui/button", "ui/spinner"},
			installed: map[string]bool{"ui/button": true, "ui/spinner": true},
			want:      nil,
		},
		{
			name:      "no peerDeps",
			peerDeps:  nil,
			installed: map[string]bool{},
			want:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &Meta{PeerDeps: tt.peerDeps}
			got := ResolvePeerDeps(meta, tt.installed)
			if len(got) != len(tt.want) {
				t.Errorf("ResolvePeerDeps() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ResolvePeerDeps()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseDep(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"lucide-react@^0.300.0", "lucide-react", "^0.300.0"},
		{"@radix-ui/react-slot@^1.0.0", "@radix-ui/react-slot", "^1.0.0"},
		{"sonner", "sonner", "latest"},
		{"react", "react", "latest"},
		{"@tanstack/react-table", "@tanstack/react-table", "latest"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, version := parseDep(tt.input)
			if name != tt.wantName {
				t.Errorf("parseDep(%q) name = %q, want %q", tt.input, name, tt.wantName)
			}
			if version != tt.wantVersion {
				t.Errorf("parseDep(%q) version = %q, want %q", tt.input, version, tt.wantVersion)
			}
		})
	}
}

func TestInjectDeps_WithVersions(t *testing.T) {
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")
	os.WriteFile(pkgPath, []byte(`{"name":"app","dependencies":{"react":"^19.0.0"}}`), 0644)

	deps := []string{"lucide-react@^0.300.0", "sonner", "@radix-ui/react-slot@^1.0.0"}
	changed, err := injectDeps(pkgPath, deps)
	if err != nil {
		t.Fatalf("injectDeps: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when new deps are injected")
	}

	data, _ := os.ReadFile(pkgPath)
	var pkg map[string]interface{}
	json.Unmarshal(data, &pkg)
	depsMap := pkg["dependencies"].(map[string]interface{})

	if depsMap["lucide-react"] != "^0.300.0" {
		t.Errorf("lucide-react = %v, want ^0.300.0", depsMap["lucide-react"])
	}
	if depsMap["sonner"] != "latest" {
		t.Errorf("sonner = %v, want latest", depsMap["sonner"])
	}
	if depsMap["@radix-ui/react-slot"] != "^1.0.0" {
		t.Errorf("@radix-ui/react-slot = %v, want ^1.0.0", depsMap["@radix-ui/react-slot"])
	}
	// Existing dep should not be overwritten
	if depsMap["react"] != "^19.0.0" {
		t.Errorf("react = %v, want ^19.0.0 (should not be overwritten)", depsMap["react"])
	}
}

func TestResolvePeerDepsRecursive_LinearChain(t *testing.T) {
	// Setup: A depends on B, B depends on C
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	regDir := filepath.Join(cacheDir, "ui")
	os.MkdirAll(regDir, 0755)

	reg := Registry{
		"ui/a": AssetInfo{
			Latest:   "1.0.0",
			Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/a/v1.0.0"}},
		},
		"ui/b": AssetInfo{
			Latest:   "1.0.0",
			Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/b/v1.0.0"}},
		},
		"ui/c": AssetInfo{
			Latest:   "1.0.0",
			Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/c/v1.0.0"}},
		},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(regDir, "registry.json"), regData, 0644)

	// Create meta files
	mkMeta := func(name string, peerDeps []string) {
		dir := filepath.Join(regDir, "ui", name, "v1.0.0")
		os.MkdirAll(dir, 0755)
		m := Meta{Name: name, Version: "1.0.0", Category: "ui", PeerDeps: peerDeps, Files: []string{name + ".tsx"}}
		data, _ := json.Marshal(m)
		os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
		os.WriteFile(filepath.Join(dir, name+".tsx"), []byte("export default null"), 0644)
	}

	mkMeta("a", []string{"ui/b"})
	mkMeta("b", []string{"ui/c"})
	mkMeta("c", nil)

	mgr := NewManager(cacheDir)
	installed := map[string]bool{}

	result, err := ResolvePeerDepsRecursive(mgr, "ui/a", installed, 5)
	if err != nil {
		t.Fatalf("ResolvePeerDepsRecursive: %v", err)
	}

	// Should return [c, b] (topological: leaf first)
	if len(result) != 2 {
		t.Fatalf("expected 2 deps, got %d: %v", len(result), result)
	}
	if result[0] != "ui/c" || result[1] != "ui/b" {
		t.Errorf("got %v, want [ui/c, ui/b]", result)
	}
}

func TestResolvePeerDepsRecursive_DiamondDependency(t *testing.T) {
	// A → B → D, A → C → D (D should appear only once)
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	regDir := filepath.Join(cacheDir, "ui")
	os.MkdirAll(regDir, 0755)

	reg := Registry{
		"ui/a": AssetInfo{Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/a/v1.0.0"}}},
		"ui/b": AssetInfo{Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/b/v1.0.0"}}},
		"ui/c": AssetInfo{Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/c/v1.0.0"}}},
		"ui/d": AssetInfo{Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/d/v1.0.0"}}},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(regDir, "registry.json"), regData, 0644)

	mkMeta := func(name string, peerDeps []string) {
		dir := filepath.Join(regDir, "ui", name, "v1.0.0")
		os.MkdirAll(dir, 0755)
		m := Meta{Name: name, Version: "1.0.0", Category: "ui", PeerDeps: peerDeps, Files: []string{name + ".tsx"}}
		data, _ := json.Marshal(m)
		os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
		os.WriteFile(filepath.Join(dir, name+".tsx"), []byte("export default null"), 0644)
	}

	mkMeta("a", []string{"ui/b", "ui/c"})
	mkMeta("b", []string{"ui/d"})
	mkMeta("c", []string{"ui/d"})
	mkMeta("d", nil)

	mgr := NewManager(cacheDir)
	result, err := ResolvePeerDepsRecursive(mgr, "ui/a", map[string]bool{}, 5)
	if err != nil {
		t.Fatalf("ResolvePeerDepsRecursive: %v", err)
	}

	// D should appear exactly once
	dCount := 0
	for _, r := range result {
		if r == "ui/d" {
			dCount++
		}
	}
	if dCount != 1 {
		t.Errorf("ui/d appears %d times, want 1. result: %v", dCount, result)
	}

	// Should have 3 items total: b, c, d
	if len(result) != 3 {
		t.Errorf("expected 3 deps, got %d: %v", len(result), result)
	}
}

func TestResolvePeerDepsRecursive_CycleDetection(t *testing.T) {
	// A → B → A (cycle)
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	regDir := filepath.Join(cacheDir, "ui")
	os.MkdirAll(regDir, 0755)

	reg := Registry{
		"ui/a": AssetInfo{Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/a/v1.0.0"}}},
		"ui/b": AssetInfo{Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/b/v1.0.0"}}},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(regDir, "registry.json"), regData, 0644)

	mkMeta := func(name string, peerDeps []string) {
		dir := filepath.Join(regDir, "ui", name, "v1.0.0")
		os.MkdirAll(dir, 0755)
		m := Meta{Name: name, Version: "1.0.0", Category: "ui", PeerDeps: peerDeps, Files: []string{name + ".tsx"}}
		data, _ := json.Marshal(m)
		os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
		os.WriteFile(filepath.Join(dir, name+".tsx"), []byte("export default null"), 0644)
	}

	mkMeta("a", []string{"ui/b"})
	mkMeta("b", []string{"ui/a"}) // cycle back to A

	mgr := NewManager(cacheDir)
	result, err := ResolvePeerDepsRecursive(mgr, "ui/a", map[string]bool{}, 5)
	if err != nil {
		t.Fatalf("ResolvePeerDepsRecursive: %v", err)
	}

	// Should only have B (A is the root, already visited)
	if len(result) != 1 || result[0] != "ui/b" {
		t.Errorf("expected [ui/b], got %v", result)
	}
}

func TestResolvePeerDepsRecursive_NoPeerDeps(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	regDir := filepath.Join(cacheDir, "ui")
	os.MkdirAll(regDir, 0755)

	reg := Registry{
		"ui/a": AssetInfo{Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {Path: "ui/a/v1.0.0"}}},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(regDir, "registry.json"), regData, 0644)

	dir := filepath.Join(regDir, "ui", "a", "v1.0.0")
	os.MkdirAll(dir, 0755)
	m := Meta{Name: "a", Version: "1.0.0", Category: "ui", Files: []string{"a.tsx"}}
	data, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)

	mgr := NewManager(cacheDir)
	result, err := ResolvePeerDepsRecursive(mgr, "ui/a", map[string]bool{}, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}
