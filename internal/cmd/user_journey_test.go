package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/lock"
)

// setupFakeAssetCache creates a minimal wk-ui and wk-api cache
// in a temp directory, avoiding any git operations.
// Uses v2 directory structure: scaffold/, ui/, components/, global/
func setupFakeAssetCache(t *testing.T) (cacheDir string) {
	t.Helper()
	cacheDir = filepath.Join(t.TempDir(), "cache")

	// --- wk-ui ---
	uiDir := filepath.Join(cacheDir, "ui")

	// scaffold/default v1.0.0
	baseDir := filepath.Join(uiDir, "scaffold", "default", "v1.0.0")
	os.MkdirAll(baseDir, 0755)
	writeTestMeta(t, baseDir, asset.Meta{
		Name: "default", Version: "1.0.0", Category: "scaffold", Dest: "site",
		Files: []string{"package.json", "embed.go"},
	})
	os.WriteFile(filepath.Join(baseDir, "package.json"), []byte(`{"name":"app","dependencies":{}}`), 0644)
	os.WriteFile(filepath.Join(baseDir, "embed.go"), []byte("package site\n"), 0644)

	// ui/button v1.0.0
	btnDir := filepath.Join(uiDir, "ui", "button", "v1.0.0")
	os.MkdirAll(btnDir, 0755)
	writeTestMeta(t, btnDir, asset.Meta{
		Name: "button", Version: "1.0.0", Category: "ui",
		Deps:  []string{"@radix-ui/react-slot"},
		Files: []string{"button.tsx"},
	})
	os.WriteFile(filepath.Join(btnDir, "button.tsx"), []byte("export const Button = () => <button/>;\n"), 0644)

	// ui/button v1.1.0 (for upgrade tests)
	btn11Dir := filepath.Join(uiDir, "ui", "button", "v1.1.0")
	os.MkdirAll(btn11Dir, 0755)
	writeTestMeta(t, btn11Dir, asset.Meta{
		Name: "button", Version: "1.1.0", Category: "ui",
		Deps:  []string{"@radix-ui/react-slot"},
		Files: []string{"button.tsx"},
	})
	os.WriteFile(filepath.Join(btn11Dir, "button.tsx"), []byte("export const Button = ({variant}) => <button data-variant={variant}/>;\n"), 0644)

	uiReg := asset.Registry{
		"scaffold/default": {Latest: "1.0.0", Versions: map[string]asset.VersionEntry{
			"1.0.0": {Path: "scaffold/default/v1.0.0"},
		}},
		"ui/button": {Latest: "1.1.0", Versions: map[string]asset.VersionEntry{
			"1.0.0": {Path: "ui/button/v1.0.0"},
			"1.1.0": {Path: "ui/button/v1.1.0"},
		}},
	}
	writeTestJSON(t, filepath.Join(uiDir, "registry.json"), uiReg)
	// .git marker so EnsureCache skips cloning
	os.MkdirAll(filepath.Join(uiDir, ".git"), 0755)

	// --- wk-api ---
	apiDir := filepath.Join(cacheDir, "api")
	userDir := filepath.Join(apiDir, "example-project", "user", "v1")
	os.MkdirAll(userDir, 0755)
	os.WriteFile(filepath.Join(userDir, "user.thrift"), []byte("service User {}"), 0644)
	os.WriteFile(filepath.Join(userDir, "user.go"), []byte("package user\n"), 0644)
	os.WriteFile(filepath.Join(userDir, "client.go"), []byte("package user // client\n"), 0644)
	os.WriteFile(filepath.Join(userDir, "client.ts"), []byte("export class UserClient {}\n"), 0644)

	apiReg := asset.Registry{
		"example-project/user": {Latest: "v1", Versions: map[string]asset.VersionEntry{
			"v1": {Path: "example-project/user/v1"},
		}},
	}
	writeTestJSON(t, filepath.Join(apiDir, "registry.json"), apiReg)
	os.MkdirAll(filepath.Join(apiDir, ".git"), 0755)

	return cacheDir
}

func writeTestMeta(t *testing.T, dir string, m asset.Meta) {
	t.Helper()
	writeTestJSON(t, filepath.Join(dir, "meta.json"), m)
}

func writeTestJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal json for %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// setupProject creates a minimal GVE project with gve.lock and site/.
func setupProject(t *testing.T, _ string) string {
	t.Helper()
	projectDir := t.TempDir()

	lf := lock.New("fake-ui-registry", "fake-api-registry")
	if err := lf.Save(filepath.Join(projectDir, "gve.lock")); err != nil {
		t.Fatal(err)
	}

	siteDir := filepath.Join(projectDir, "site")
	os.MkdirAll(siteDir, 0755)
	os.WriteFile(filepath.Join(siteDir, "package.json"), []byte(`{"name":"app","dependencies":{}}`), 0644)
	os.MkdirAll(filepath.Join(siteDir, "src", "shared", "wk", "ui"), 0755)
	os.MkdirAll(filepath.Join(siteDir, "src", "shared", "wk", "components"), 0755)

	return projectDir
}

// chdir changes to dir for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

// ─── Test: Full User Journey ─────────────────────────────────────────

func TestUserJourney(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	chdir(t, projectDir)

	mgr := asset.NewManager(cacheDir)

	t.Run("Step1_UIAdd_Button", func(t *testing.T) {
		ver, err := asset.InstallUIAsset(mgr, "ui/button", "1.0.0", projectDir)
		if err != nil {
			t.Fatalf("InstallUIAsset: %v", err)
		}
		if ver != "1.0.0" {
			t.Errorf("version = %q, want 1.0.0", ver)
		}

		btnFile := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "button.tsx")
		if _, err := os.Stat(btnFile); err != nil {
			t.Errorf("button.tsx not installed: %v", err)
		}

		lf, _ := lock.Load(filepath.Join(projectDir, "gve.lock"))
		lf.SetUIAsset("ui/button", ver)
		lf.Save(filepath.Join(projectDir, "gve.lock"))
	})

	t.Run("Step2_UIList_ShowsButton", func(t *testing.T) {
		lf, err := lock.Load(filepath.Join(projectDir, "gve.lock"))
		if err != nil {
			t.Fatal(err)
		}
		v, ok := lf.GetUIAsset("ui/button")
		if !ok {
			t.Fatal("ui/button not in gve.lock")
		}
		if v != "1.0.0" {
			t.Errorf("button version = %q, want 1.0.0", v)
		}
	})

	t.Run("Step3_APIAdd_User", func(t *testing.T) {
		ver, err := asset.InstallAPIAsset(mgr, "example-project/user", "v1", projectDir)
		if err != nil {
			t.Fatalf("InstallAPIAsset: %v", err)
		}
		if ver != "v1" {
			t.Errorf("version = %q, want v1", ver)
		}

		expectedFiles := []string{"user.thrift"}
		for _, f := range expectedFiles {
			p := filepath.Join(projectDir, "api", "example-project", "user", "v1", f)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("expected api file %s: %v", f, err)
			}
		}
		for _, f := range []string{"user.go", "client.go", "client.ts"} {
			p := filepath.Join(projectDir, "api", "example-project", "user", "v1", f)
			if _, err := os.Stat(p); err == nil {
				t.Errorf("did not expect generated api file in root api dir: %s", f)
			}
		}

		lf, _ := lock.Load(filepath.Join(projectDir, "gve.lock"))
		lf.SetAPIAsset("example-project/user", ver)
		lf.Save(filepath.Join(projectDir, "gve.lock"))
	})

	t.Run("Step4_Status_CheckVersions", func(t *testing.T) {
		lf, _ := lock.Load(filepath.Join(projectDir, "gve.lock"))
		uiReg, _ := mgr.GetRegistry("ui")
		apiReg, _ := mgr.GetRegistry("api")

		// ui/button: 1.0.0 installed, 1.1.0 available
		btnInfo := uiReg["ui/button"]
		btnLock := lf.UI.Assets["ui/button"]
		if btnLock.Version == btnInfo.Latest {
			t.Error("button should have an update available")
		}

		// api user: v1 installed, v1 is latest
		apiInfo := apiReg["example-project/user"]
		apiLock := lf.API.Assets["example-project/user"]
		if apiLock.Version != apiInfo.Latest {
			t.Errorf("api user version %q != latest %q", apiLock.Version, apiInfo.Latest)
		}
	})

	t.Run("Step5_UIDiff_NoChanges", func(t *testing.T) {
		reg, _ := mgr.GetRegistry("ui")
		ve := reg["ui/button"].Versions["1.0.0"]
		cacheAssetDir := mgr.GetAssetDir("ui", ve.Path)
		meta, _ := asset.LoadMeta(filepath.Join(cacheAssetDir, "meta.json"))

		localDir := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui")
		diffs, err := asset.DiffAsset(localDir, cacheAssetDir, meta.Files)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range diffs {
			if d.Status != "unchanged" {
				t.Errorf("file %s status = %q, want unchanged", d.File, d.Status)
			}
		}
	})

	t.Run("Step6_UIDiff_WithLocalEdit", func(t *testing.T) {
		btnFile := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "button.tsx")
		os.WriteFile(btnFile, []byte("// customized\nexport const Button = () => <button className='custom'/>;\n"), 0644)

		reg, _ := mgr.GetRegistry("ui")
		ve := reg["ui/button"].Versions["1.0.0"]
		cacheAssetDir := mgr.GetAssetDir("ui", ve.Path)
		meta, _ := asset.LoadMeta(filepath.Join(cacheAssetDir, "meta.json"))

		localDir := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui")
		diffs, _ := asset.DiffAsset(localDir, cacheAssetDir, meta.Files)

		found := false
		for _, d := range diffs {
			if d.File == "button.tsx" && d.Status == "modified" {
				found = true
				if d.Diff == "" {
					t.Error("expected non-empty diff for modified file")
				}
			}
		}
		if !found {
			t.Error("expected button.tsx to be marked as modified")
		}

		if !asset.HasLocalChanges(localDir, cacheAssetDir, meta.Files) {
			t.Error("HasLocalChanges should return true")
		}
	})

	t.Run("Step7_Sync_RestoresMissing", func(t *testing.T) {
		btnFile := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "button.tsx")
		os.Remove(btnFile)

		// Reinstall button from lock
		lf, _ := lock.Load(filepath.Join(projectDir, "gve.lock"))
		v := lf.UI.Assets["ui/button"].Version
		_, err := asset.InstallUIAsset(mgr, "ui/button", v, projectDir)
		if err != nil {
			t.Fatalf("reinstall button: %v", err)
		}

		if _, err := os.Stat(btnFile); err != nil {
			t.Errorf("button.tsx not restored: %v", err)
		}
	})

	t.Run("Step8_Sync_API_RestoresMissing", func(t *testing.T) {
		apiDir := filepath.Join(projectDir, "api", "example-project", "user", "v1")
		os.RemoveAll(apiDir)

		if asset.APIAssetDirExists(projectDir, "example-project/user", "v1") {
			t.Fatal("api dir should not exist after removal")
		}

		lf, _ := lock.Load(filepath.Join(projectDir, "gve.lock"))
		v := lf.API.Assets["example-project/user"].Version
		_, err := asset.InstallAPIAsset(mgr, "example-project/user", v, projectDir)
		if err != nil {
			t.Fatalf("reinstall api: %v", err)
		}

		if !asset.APIAssetDirExists(projectDir, "example-project/user", "v1") {
			t.Error("api dir should exist after reinstall")
		}
	})

	t.Run("Step9_UIUpgrade_ToLatest", func(t *testing.T) {
		ver, err := asset.InstallUIAsset(mgr, "ui/button", "1.1.0", projectDir)
		if err != nil {
			t.Fatalf("upgrade button: %v", err)
		}
		if ver != "1.1.0" {
			t.Errorf("version = %q, want 1.1.0", ver)
		}

		btnFile := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "button.tsx")
		data, _ := os.ReadFile(btnFile)
		if !strings.Contains(string(data), "variant") {
			t.Error("button.tsx should contain v1.1.0 content with 'variant'")
		}

		lf, _ := lock.Load(filepath.Join(projectDir, "gve.lock"))
		lf.SetUIAsset("ui/button", ver)
		lf.Save(filepath.Join(projectDir, "gve.lock"))

		v, _ := lf.GetUIAsset("ui/button")
		if v != "1.1.0" {
			t.Errorf("lock version = %q, want 1.1.0", v)
		}
	})

	t.Run("Step10_APIPush_DryRun", func(t *testing.T) {
		// Create a local thrift file to push
		localThriftDir := filepath.Join(projectDir, "api", "test-project", "widget", "v1")
		os.MkdirAll(localThriftDir, 0755)
		os.WriteFile(filepath.Join(localThriftDir, "widget.thrift"), []byte("service Widget { string getWidget() }"), 0644)

		apiCacheDir := filepath.Join(cacheDir, "api")

		opts := asset.APIPushOptions{
			CacheDir:  apiCacheDir,
			Project:   "test-project",
			Resource:  "widget",
			Version:   "v1",
			SourceDir: localThriftDir,
			DryRun:    true,
		}

		if err := asset.PushAPIToRegistry(opts); err != nil {
			t.Fatalf("APIPush dry-run: %v", err)
		}

		// Verify nothing was written to cache
		destDir := filepath.Join(apiCacheDir, "test-project", "widget", "v1")
		if _, err := os.Stat(destDir); err == nil {
			t.Error("dry-run should not create destination directory in cache")
		}
	})

	t.Run("Step11_APIBuildRegistry", func(t *testing.T) {
		apiCacheDir := filepath.Join(cacheDir, "api")

		reg, err := asset.BuildAPIRegistry(apiCacheDir)
		if err != nil {
			t.Fatalf("BuildAPIRegistry: %v", err)
		}

		info, ok := reg["example-project/user"]
		if !ok {
			t.Fatal("example-project/user not in built registry")
		}
		if info.Latest != "v1" {
			t.Errorf("latest = %q, want v1", info.Latest)
		}
	})
}

// ─── Test: parseAPIAssetArg ──────────────────────────────────────────

func TestParseAPIAssetArg(t *testing.T) {
	tests := []struct {
		arg          string
		wantResource string
		wantVersion  string
	}{
		{"ai-worker/task@v1", "ai-worker/task", "v1"},
		{"ai-worker/task", "ai-worker/task", ""},
		{"example-project/user@v2", "example-project/user", "v2"},
		{"single", "single", ""},
		{"deep/nested/path@v3", "deep/nested/path", "v3"},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			resource, version := parseAPIAssetArg(tt.arg)
			if resource != tt.wantResource {
				t.Errorf("resource = %q, want %q", resource, tt.wantResource)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

// ─── Test: doctor helper functions ───────────────────────────────────

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"go version go1.22.3 linux/amd64", "1.22.3"},
		{"v22.13.1", "22.13.1"},
		{"git version 2.43.7", "2.43.7"},
		{"10.28.2", "10.28.2"},
		{"no version here", "no version here"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractVersion(tt.input)
			if got != tt.want {
				t.Errorf("extractVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseGoVersion(t *testing.T) {
	tests := []struct {
		ver       string
		wantMajor int
		wantMinor int
	}{
		{"1.22.3", 1, 22},
		{"1.21.0", 1, 21},
		{"2.0.0", 2, 0},
		{"1.18", 1, 18},
	}
	for _, tt := range tests {
		t.Run(tt.ver, func(t *testing.T) {
			major, minor := parseGoVersion(tt.ver)
			if major != tt.wantMajor || minor != tt.wantMinor {
				t.Errorf("parseGoVersion(%q) = (%d, %d), want (%d, %d)",
					tt.ver, major, minor, tt.wantMajor, tt.wantMinor)
			}
		})
	}
}

func TestParseNodeMajor(t *testing.T) {
	tests := []struct {
		ver  string
		want int
	}{
		{"v22.13.1", 22},
		{"v18.0.0", 18},
		{"20.5.1", 20},
		{"v16.0.0", 16},
	}
	for _, tt := range tests {
		t.Run(tt.ver, func(t *testing.T) {
			got := parseNodeMajor(tt.ver)
			if got != tt.want {
				t.Errorf("parseNodeMajor(%q) = %d, want %d", tt.ver, got, tt.want)
			}
		})
	}
}

// ─── Test: RegistryBuild flow ────────────────────────────────────────

func TestRegistryBuild_MultiVersion(t *testing.T) {
	dir := t.TempDir()

	// Create v2 structure: ui/widget/v{x}/
	for _, v := range []string{"v1.0.0", "v2.0.0", "v1.5.0"} {
		d := filepath.Join(dir, "ui", "widget", v)
		os.MkdirAll(d, 0755)
		ver := strings.TrimPrefix(v, "v")
		writeTestMeta(t, d, asset.Meta{
			Name: "widget", Version: ver, Category: "ui", Files: []string{"widget.tsx"},
		})
		os.WriteFile(filepath.Join(d, "widget.tsx"), []byte("export const Widget = () => null;\n"), 0644)
	}

	reg2, _, err := asset.BuildRegistryV2(dir)
	if err != nil {
		t.Fatalf("BuildRegistryV2: %v", err)
	}
	if err := asset.WriteRegistryV2(reg2, filepath.Join(dir, "registry.json")); err != nil {
		t.Fatalf("WriteRegistryV2: %v", err)
	}

	reg, err := asset.LoadRegistry(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}

	info, ok := reg["ui/widget"]
	if !ok {
		t.Fatal("ui/widget not in registry")
	}
	if info.Latest != "2.0.0" {
		t.Errorf("latest = %q, want 2.0.0", info.Latest)
	}
	if len(info.Versions) != 3 {
		t.Errorf("version count = %d, want 3", len(info.Versions))
	}
}

// ─── Test: Global asset install with dest ────────────────────────────

func TestInstallGlobalAsset(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)

	mgr := asset.NewManager(cacheDir)

	// scaffold/default has dest: "site", should install to site/
	ver, err := asset.InstallUIAsset(mgr, "scaffold/default", "1.0.0", projectDir)
	if err != nil {
		t.Fatalf("install scaffold/default: %v", err)
	}
	if ver != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", ver)
	}

	embedFile := filepath.Join(projectDir, "site", "embed.go")
	if _, err := os.Stat(embedFile); err != nil {
		t.Errorf("embed.go not installed to site/: %v", err)
	}

	pkgFile := filepath.Join(projectDir, "site", "package.json")
	if _, err := os.Stat(pkgFile); err != nil {
		t.Errorf("package.json not installed to site/: %v", err)
	}
}

// ─── Test: NPM deps injection ────────────────────────────────────────

func TestUIAdd_InjectsDeps(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)

	mgr := asset.NewManager(cacheDir)

	_, err := asset.InstallUIAsset(mgr, "ui/button", "1.0.0", projectDir)
	if err != nil {
		t.Fatal(err)
	}

	pkgData, err := os.ReadFile(filepath.Join(projectDir, "site", "package.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(pkgData), "@radix-ui/react-slot") {
		t.Error("package.json should contain @radix-ui/react-slot after button install")
	}
}

// ─── Test: printAssetStatus ──────────────────────────────────────────

func TestPrintAssetStatus(t *testing.T) {
	assets := map[string]lock.AssetEntry{
		"ui/button":             {Version: "1.0.0"},
		"components/data-table": {Version: "2.0.0"},
		"missing":               {Version: "1.0.0"},
	}
	reg := asset.Registry{
		"ui/button":             {Latest: "1.1.0", Versions: map[string]asset.VersionEntry{"1.0.0": {Path: "x"}}},
		"components/data-table": {Latest: "2.0.0", Versions: map[string]asset.VersionEntry{"2.0.0": {Path: "y"}}},
	}

	// printAssetStatus just prints — verify it doesn't panic
	printAssetStatus(assets, reg)
}

// ─── Test: assetExists edge cases ────────────────────────────────────

func TestAssetExists_NotInRegistry(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)

	reg := asset.Registry{}

	if assetExists("nonexistent", "1.0.0", reg, mgr, projectDir) {
		t.Error("assetExists should return false for asset not in registry")
	}
}

func TestAssetExists_VersionNotInRegistry(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)

	reg := asset.Registry{
		"ui/button": {Latest: "1.0.0", Versions: map[string]asset.VersionEntry{
			"1.0.0": {Path: "ui/button/v1.0.0"},
		}},
	}

	if assetExists("ui/button", "9.9.9", reg, mgr, projectDir) {
		t.Error("assetExists should return false for version not in registry")
	}
}

func TestAssetExists_FilePresent(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)
	reg, _ := mgr.GetRegistry("ui")

	// Install button first
	asset.InstallUIAsset(mgr, "ui/button", "1.0.0", projectDir)

	if !assetExists("ui/button", "1.0.0", reg, mgr, projectDir) {
		t.Error("assetExists should return true when files are present")
	}
}

func TestAssetExists_FileMissing(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)
	reg, _ := mgr.GetRegistry("ui")

	// button not installed — files missing
	if assetExists("ui/button", "1.0.0", reg, mgr, projectDir) {
		t.Error("assetExists should return false when files are missing")
	}
}

// ─── Test: resolveAssetInfo ───────────────────────────────────────────

func TestResolveAssetInfo(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	mgr := asset.NewManager(cacheDir)
	reg, _ := mgr.GetRegistry("ui")

	t.Run("scaffold asset with dest", func(t *testing.T) {
		got, _ := resolveAssetInfo("scaffold/default", "1.0.0", reg, mgr)
		if got != "site/" {
			t.Errorf("resolveAssetInfo(scaffold/default) dest = %q, want \"site/\"", got)
		}
	})

	t.Run("ui asset without dest", func(t *testing.T) {
		got, _ := resolveAssetInfo("ui/button", "1.0.0", reg, mgr)
		if got != "site/src/shared/wk/ui/" {
			t.Errorf("resolveAssetInfo(ui/button) dest = %q, want \"site/src/shared/wk/ui/\"", got)
		}
	})

	t.Run("unknown asset", func(t *testing.T) {
		got, _ := resolveAssetInfo("nonexistent", "1.0.0", reg, mgr)
		if got != "site/src/shared/wk/ui/" {
			t.Errorf("resolveAssetInfo(nonexistent) dest = %q, want fallback path", got)
		}
	})

	t.Run("nil registry", func(t *testing.T) {
		got, _ := resolveAssetInfo("ui/button", "1.0.0", nil, mgr)
		if got != "site/src/shared/wk/ui/" {
			t.Errorf("resolveAssetInfo with nil reg dest = %q, want fallback path", got)
		}
	})

	t.Run("description returned", func(t *testing.T) {
		_, desc := resolveAssetInfo("ui/button", "1.0.0", reg, mgr)
		// Description may be empty in test fixtures, just ensure no error
		_ = desc
	})
}

// ─── Test: checkGveLock ──────────────────────────────────────────────

func TestCheckGveLock_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "gve.lock"), []byte("{invalid json"), 0644)

	chdir(t, dir)

	result := checkGveLock()
	if result.ok {
		t.Error("checkGveLock should fail for invalid JSON")
	}
	if !strings.Contains(result.message, "invalid") {
		t.Errorf("message = %q, should contain 'invalid'", result.message)
	}
}

func TestCheckGveLock_Valid(t *testing.T) {
	dir := t.TempDir()
	lf := lock.New("ui-reg", "api-reg")
	lf.Save(filepath.Join(dir, "gve.lock"))

	chdir(t, dir)

	result := checkGveLock()
	if !result.ok {
		t.Errorf("checkGveLock should pass for valid lock, got: %s", result.message)
	}
	if result.version != "valid" {
		t.Errorf("version = %q, want valid", result.version)
	}
}

func TestCheckGveLock_NotInProject(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	result := checkGveLock()
	if !result.ok {
		t.Error("checkGveLock should be ok when not in a GVE project")
	}
}

// ─── Test: Idempotent install ────────────────────────────────────────

func TestUIInstall_Idempotent(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)

	// Install twice
	v1, err := asset.InstallUIAsset(mgr, "ui/button", "1.0.0", projectDir)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := asset.InstallUIAsset(mgr, "ui/button", "1.0.0", projectDir)
	if err != nil {
		t.Fatal(err)
	}

	if v1 != v2 {
		t.Errorf("idempotent install: v1=%q, v2=%q", v1, v2)
	}

	btnFile := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "button.tsx")
	data, _ := os.ReadFile(btnFile)
	if len(data) == 0 {
		t.Error("button.tsx should have content after double install")
	}
}

// ─── Test: Install latest when version is empty ──────────────────────

func TestUIInstall_LatestWhenEmpty(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)

	ver, err := asset.InstallUIAsset(mgr, "ui/button", "", projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if ver != "1.1.0" {
		t.Errorf("empty version should install latest, got %q", ver)
	}
}

func TestAPIInstall_LatestWhenEmpty(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)

	ver, err := asset.InstallAPIAsset(mgr, "example-project/user", "", projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if ver != "v1" {
		t.Errorf("empty version should install latest, got %q", ver)
	}
}

// ─── Test: Error paths ───────────────────────────────────────────────

func TestUIInstall_NonexistentAsset(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)

	_, err := asset.InstallUIAsset(mgr, "does-not-exist", "1.0.0", projectDir)
	if err == nil {
		t.Error("expected error for nonexistent asset")
	}
}

func TestAPIInstall_NonexistentResource(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)

	_, err := asset.InstallAPIAsset(mgr, "fake-project/fake-resource", "v1", projectDir)
	if err == nil {
		t.Error("expected error for nonexistent API resource")
	}
}

// ─── Test: Diff after file deletion (added status) ───────────────────

func TestDiffAsset_DeletedFile(t *testing.T) {
	cacheDir := setupFakeAssetCache(t)
	projectDir := setupProject(t, cacheDir)
	mgr := asset.NewManager(cacheDir)

	// Install button, then delete a file
	asset.InstallUIAsset(mgr, "ui/button", "1.0.0", projectDir)

	btnFile := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "button.tsx")
	os.Remove(btnFile)

	reg, _ := mgr.GetRegistry("ui")
	ve := reg["ui/button"].Versions["1.0.0"]
	cacheAssetDir := mgr.GetAssetDir("ui", ve.Path)
	meta, _ := asset.LoadMeta(filepath.Join(cacheAssetDir, "meta.json"))

	localDir := filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", "button")
	diffs, err := asset.DiffAsset(localDir, cacheAssetDir, meta.Files)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, d := range diffs {
		if d.File == "button.tsx" && d.Status == "deleted" {
			found = true
		}
	}
	if !found {
		t.Error("expected button.tsx to be marked as deleted")
	}
}
