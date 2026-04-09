# GVE v2 UI Asset Architecture Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade GVE CLI and wk-ui registry from v1 flat-asset structure to v2 category-based structure (scaffold/ui/components/global), with new lock format, category-aware install paths, peerDeps resolution, and CSS validation.

**Architecture:** Bottom-up approach: data layer first (meta, lock, registry), then asset operations (builder, installer), then CLI commands (registry build, ui add, ui list, ui diff, ui sync, init), and finally wk-ui remote repo rebuild. Each phase produces independently testable code with all existing tests kept green.

**Tech Stack:** Go 1.22+, cobra CLI, gh CLI (for wk-ui remote ops), standard library JSON/os/filepath.

**Spec Document:** `/data/home/castlexu/github/gve/2026-03-17-gve-v2-ui-asset-architecture.md`

---

## Phase 1: Data Layer (meta.go, lock.go, registry.go)

Foundation types that everything else depends on. No CLI changes yet, all existing tests must still pass.

### Task 1.1: Upgrade Meta struct with v2 fields

**Files:**
- Modify: `internal/asset/meta.go`
- Create: `internal/asset/meta_test.go`

- [ ] **Step 1: Write failing tests for new Meta fields**

```go
// internal/asset/meta_test.go
package asset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMeta_V2Fields(t *testing.T) {
	dir := t.TempDir()
	metaJSON := `{
		"$schema": "https://gve.dev/schema/meta.json",
		"name": "data-table",
		"version": "2.0.0",
		"category": "component",
		"description": "A data table with sorting and pagination.",
		"dest": "",
		"deps": ["@tanstack/react-table"],
		"peerDeps": ["ui/button", "ui/spinner"],
		"files": ["data-table.tsx"]
	}`
	path := filepath.Join(dir, "meta.json")
	os.WriteFile(path, []byte(metaJSON), 0644)

	m, err := LoadMeta(path)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if m.Category != "component" {
		t.Errorf("Category = %q, want %q", m.Category, "component")
	}
	if m.Description != "A data table with sorting and pagination." {
		t.Errorf("Description = %q", m.Description)
	}
	if len(m.PeerDeps) != 2 || m.PeerDeps[0] != "ui/button" {
		t.Errorf("PeerDeps = %v", m.PeerDeps)
	}
}

func TestLoadMeta_V1Compat(t *testing.T) {
	dir := t.TempDir()
	metaJSON := `{"name":"button","version":"1.0.0","deps":[],"files":["button.tsx"]}`
	path := filepath.Join(dir, "meta.json")
	os.WriteFile(path, []byte(metaJSON), 0644)

	m, err := LoadMeta(path)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if m.Category != "" {
		t.Errorf("Category should be empty for v1, got %q", m.Category)
	}
	if m.PeerDeps != nil {
		t.Errorf("PeerDeps should be nil for v1, got %v", m.PeerDeps)
	}
}

func TestInferCategory(t *testing.T) {
	tests := []struct {
		path     string
		want     string
	}{
		{"scaffold/default/v1.0.0", "scaffold"},
		{"ui/spinner/v1.0.0", "ui"},
		{"components/data-table/v2.0.0", "component"},
		{"global/theme/v1.0.0", "global"},
		{"assets/button/v1.0.0", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := InferCategory(tt.path)
			if got != tt.want {
				t.Errorf("InferCategory(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestMeta_MarshalRoundTrip(t *testing.T) {
	m := Meta{
		Schema:      "https://gve.dev/schema/meta.json",
		Name:        "spinner",
		Version:     "1.0.0",
		Category:    "ui",
		Description: "Loading spinner.",
		Deps:        []string{},
		PeerDeps:    []string{},
		Files:       []string{"spinner.tsx"},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var m2 Meta
	json.Unmarshal(data, &m2)
	if m2.Schema != m.Schema || m2.Category != m.Category {
		t.Errorf("roundtrip mismatch: %+v", m2)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/asset/ -run "TestLoadMeta_V2|TestInferCategory|TestMeta_Marshal" -v`
Expected: FAIL (Category, Description, PeerDeps, Schema fields don't exist)

- [ ] **Step 3: Update Meta struct and add InferCategory**

```go
// internal/asset/meta.go
package asset

import (
	"encoding/json"
	"os"
	"strings"
)

type Meta struct {
	Schema      string   `json:"$schema,omitempty"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Category    string   `json:"category,omitempty"`
	Description string   `json:"description,omitempty"`
	Dest        string   `json:"dest,omitempty"`
	Deps        []string `json:"deps,omitempty"`
	PeerDeps    []string `json:"peerDeps,omitempty"`
	Files       []string `json:"files"`
}

func LoadMeta(path string) (*Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// InferCategory derives the asset category from its registry path.
// e.g. "ui/spinner/v1.0.0" -> "ui", "components/data-table/v2.0.0" -> "component"
func InferCategory(assetPath string) string {
	if assetPath == "" {
		return ""
	}
	parts := strings.SplitN(assetPath, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	switch parts[0] {
	case "scaffold":
		return "scaffold"
	case "ui":
		return "ui"
	case "components":
		return "component"
	case "global":
		return "global"
	default:
		return ""
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/asset/ -run "TestLoadMeta_V2|TestInferCategory|TestMeta_Marshal" -v`
Expected: PASS

- [ ] **Step 5: Run ALL tests to verify nothing broke**

Run: `go test ./... -count=1`
Expected: All PASS (existing tests are unaffected since new fields are `omitempty`)

- [ ] **Step 6: Commit**

```bash
git add internal/asset/meta.go internal/asset/meta_test.go
git commit -m "feat(asset): add v2 Meta fields (category, description, peerDeps, schema)"
```

---

### Task 1.2: Upgrade lock.go with v2 format and v1 migration

**Files:**
- Modify: `internal/lock/lock.go`
- Modify: `internal/lock/lock_test.go`

- [ ] **Step 1: Write failing tests for v2 lock and migration**

Add to `internal/lock/lock_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/lock/ -run "TestNewV2|TestLoadV1_AutoMigrate|TestLoadV2_NoMigration" -v`
Expected: FAIL

- [ ] **Step 3: Implement v2 lock with migration**

Update `internal/lock/lock.go`:

```go
package lock

import (
	"encoding/json"
	"os"
	"strings"
)

type AssetEntry struct {
	Version string `json:"version"`
}

type AssetGroup struct {
	Registry string                `json:"registry"`
	Assets   map[string]AssetEntry `json:"assets"`
}

type LockFile struct {
	Version string     `json:"version"`
	UI      AssetGroup `json:"ui"`
	API     AssetGroup `json:"api"`
}

func New(uiRegistry, apiRegistry string) *LockFile {
	return &LockFile{
		Version: "2",
		UI: AssetGroup{
			Registry: uiRegistry,
			Assets:   make(map[string]AssetEntry),
		},
		API: AssetGroup{
			Registry: apiRegistry,
			Assets:   make(map[string]AssetEntry),
		},
	}
}

func Load(path string) (*LockFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lf LockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, err
	}
	if lf.UI.Assets == nil {
		lf.UI.Assets = make(map[string]AssetEntry)
	}
	if lf.API.Assets == nil {
		lf.API.Assets = make(map[string]AssetEntry)
	}

	if lf.Version == "1" || lf.Version == "" {
		migrateV1ToV2(&lf)
	}

	return &lf, nil
}

// migrateV1ToV2 converts v1 lock keys to v2 category-prefixed keys.
func migrateV1ToV2(lf *LockFile) {
	migrated := make(map[string]AssetEntry, len(lf.UI.Assets))
	for key, entry := range lf.UI.Assets {
		newKey := migrateAssetKey(key)
		migrated[newKey] = entry
	}
	lf.UI.Assets = migrated
	lf.Version = "2"
}

// migrateAssetKey converts a v1 bare key to a v2 category-prefixed key.
func migrateAssetKey(key string) string {
	// Already has a category prefix
	if strings.Contains(key, "/") {
		return key
	}
	// Special case: base-setup -> scaffold/default
	if key == "base-setup" {
		return "scaffold/default"
	}
	// Default: assume ui/ prefix
	return "ui/" + key
}

func (lf *LockFile) Save(path string) error {
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func (lf *LockFile) SetUIAsset(name, version string) {
	lf.UI.Assets[name] = AssetEntry{Version: version}
}

func (lf *LockFile) SetAPIAsset(name, version string) {
	lf.API.Assets[name] = AssetEntry{Version: version}
}

func (lf *LockFile) GetUIAsset(name string) (string, bool) {
	entry, ok := lf.UI.Assets[name]
	if !ok {
		return "", false
	}
	return entry.Version, true
}

func (lf *LockFile) GetAPIAsset(name string) (string, bool) {
	entry, ok := lf.API.Assets[name]
	if !ok {
		return "", false
	}
	return entry.Version, true
}
```

- [ ] **Step 4: Run new tests**

Run: `go test ./internal/lock/ -v`
Expected: PASS

- [ ] **Step 5: Fix existing tests that expect version "1"**

The existing `TestNewAndSaveLoad` expects `loaded.Version == "1"`. Update it to expect `"2"`. Also update key expectations from `"button"` to `"ui/button"` since Load now migrates.

In `internal/lock/lock_test.go`, update `TestNewAndSaveLoad`:
- Change `loaded.Version != "1"` to `loaded.Version != "2"`
- Change `loaded.GetUIAsset("button")` — since New() creates v2, keys set via SetUIAsset should still work as-is (the test sets `"button"` which won't be migrated since it's created with v2). However, `SetUIAsset("button", ...)` will store `"button"` in v2 format. The test's key will still work. But we should update to use v2-style keys for clarity:

```go
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
```

- [ ] **Step 6: Run ALL tests**

Run: `go test ./... -count=1`
Expected: WILL FAIL — integration tests in `internal/cmd/` use v1 keys ("button", "base-setup"). We'll fix those in Task 1.3.

- [ ] **Step 7: Commit lock changes**

```bash
git add internal/lock/lock.go internal/lock/lock_test.go
git commit -m "feat(lock): upgrade to v2 format with auto-migration from v1"
```

---

### Task 1.3: Upgrade registry.go to support v2 format + fix integration tests

**Files:**
- Modify: `internal/asset/registry.go`
- Modify: `internal/asset/registry_test.go`
- Modify: `internal/cmd/user_journey_test.go` (update fixture keys to v2)

- [ ] **Step 1: Write failing tests for v2 registry**

Add to `internal/asset/registry_test.go`:

```go
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
	if _, ok := reg["ui/spinner"]; !ok {
		t.Error("ui/spinner not found in registry")
	}
	if _, ok := reg["scaffold/default"]; !ok {
		t.Error("scaffold/default not found in registry")
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
		{"spinner", "ui/spinner", true},       // shortname -> ui/ first
		{"data-table", "components/data-table", true}, // shortname -> components/
		{"theme", "global/theme", true},       // shortname -> global/
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
		"scaffold/default":     {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {}}},
		"ui/spinner":           {Latest: "1.0.0", Versions: map[string]VersionEntry{"1.0.0": {}}},
		"components/data-table":{Latest: "2.0.0", Versions: map[string]VersionEntry{"2.0.0": {}}},
	}

	scaffolds := reg.ListByCategory("scaffold")
	if len(scaffolds) != 1 || scaffolds[0] != "scaffold/default" {
		t.Errorf("scaffolds = %v", scaffolds)
	}
	uis := reg.ListByCategory("ui")
	if len(uis) != 1 {
		t.Errorf("uis = %v", uis)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/asset/ -run "TestLoadRegistry_V2|TestRegistry_Resolve|TestRegistry_ListByCategory" -v`
Expected: FAIL

- [ ] **Step 3: Implement v2 registry support**

Update `internal/asset/registry.go` — the key change is: `LoadRegistry` must handle the top-level `$schema` and `version` fields that aren't asset entries. Also add `ResolveAssetName` and `ListByCategory`.

```go
package asset

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/castle-x/gve/internal/semver"
)

type VersionEntry struct {
	Path string `json:"path"`
}

type AssetInfo struct {
	Latest   string                  `json:"latest"`
	Versions map[string]VersionEntry `json:"versions"`
}

// Registry maps asset names to their version info.
type Registry map[string]AssetInfo

// LoadRegistry loads a registry.json, ignoring top-level meta fields ($schema, version).
func LoadRegistry(path string) (Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	// First unmarshal into raw map to filter out non-asset keys
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	reg := make(Registry)
	for key, val := range raw {
		// Skip top-level meta fields
		if key == "$schema" || key == "version" {
			continue
		}
		var info AssetInfo
		if err := json.Unmarshal(val, &info); err != nil {
			continue // skip malformed entries
		}
		if info.Versions != nil {
			reg[key] = info
		}
	}

	return reg, nil
}

func (r Registry) GetLatest(assetName string) (version string, path string, err error) {
	info, ok := r[assetName]
	if !ok {
		return "", "", fmt.Errorf("asset %q not found in registry", assetName)
	}
	entry, ok := info.Versions[info.Latest]
	if !ok {
		return "", "", fmt.Errorf("latest version %s of %q not found", info.Latest, assetName)
	}
	return info.Latest, entry.Path, nil
}

func (r Registry) GetVersion(assetName, version string) (string, error) {
	info, ok := r[assetName]
	if !ok {
		return "", fmt.Errorf("asset %q not found in registry", assetName)
	}

	available := make([]string, 0, len(info.Versions))
	for v := range info.Versions {
		available = append(available, v)
	}

	resolved, err := semver.ResolveVersion(version, available)
	if err != nil {
		return "", fmt.Errorf("resolve version for %q: %w", assetName, err)
	}

	entry, ok := info.Versions[resolved]
	if !ok {
		return "", fmt.Errorf("version %s of %q not found", resolved, assetName)
	}
	return entry.Path, nil
}

func (r Registry) ListAssets() []string {
	names := make([]string, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveAssetName resolves a potentially short name to a full category/name key.
// Priority: exact match -> ui/{name} -> components/{name} -> global/{name}.
func (r Registry) ResolveAssetName(name string) (string, bool) {
	// Exact match (already has category prefix)
	if _, ok := r[name]; ok {
		return name, true
	}

	// Try shortname resolution
	for _, prefix := range []string{"ui/", "components/", "global/"} {
		fullName := prefix + name
		if _, ok := r[fullName]; ok {
			return fullName, true
		}
	}

	return "", false
}

// ListByCategory returns all asset names matching the given category prefix.
func (r Registry) ListByCategory(category string) []string {
	prefix := category + "/"
	var names []string
	for name := range r {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
```

- [ ] **Step 4: Run registry tests**

Run: `go test ./internal/asset/ -run "TestLoadRegistry|TestRegistry_Resolve|TestRegistry_ListByCategory" -v`
Expected: PASS

- [ ] **Step 5: Update integration test fixtures to v2 format**

Update `internal/cmd/user_journey_test.go`:

1. `setupFakeAssetCache`: change registry keys from `"base-setup"` to `"scaffold/default"`, `"button"` to `"ui/button"`. Change asset paths from `"assets/..."` to `"scaffold/..."`, `"ui/..."`. Change directory structure accordingly.

2. `setupProject`: change placeholder dir from `shared/ui` to `shared/wk/ui`.

3. All test steps: change lock keys from `"button"` to `"ui/button"`, `"base-setup"` to `"scaffold/default"`. Change install paths from `shared/ui/button` to `shared/wk/ui/button`.

4. `resolveDestPath` and `assetExists` tests: update to v2 keys and paths.

(Full code is large — the developer should update each reference systematically. The key mapping is:)

| v1 key | v2 key | v1 path | v2 path | v1 install dir | v2 install dir |
|--------|--------|---------|---------|----------------|----------------|
| `base-setup` | `scaffold/default` | `assets/base-setup/v1.0.0` | `scaffold/default/v1.0.0` | `site/` | `site/` |
| `button` | `ui/button` | `assets/button/v1.0.0` | `ui/button/v1.0.0` | `site/src/shared/ui/button/` | `site/src/shared/wk/ui/button/` |

- [ ] **Step 6: Run ALL tests**

Run: `go test ./... -count=1`
Expected: PASS (may still fail until installer paths are updated in Phase 2)

- [ ] **Step 7: Commit**

```bash
git add internal/asset/registry.go internal/asset/registry_test.go internal/cmd/user_journey_test.go
git commit -m "feat(registry): support v2 format with category prefixes and shortname resolution"
```

---

## Phase 2: Asset Operations (builder.go, installer.go)

### Task 2.1: Upgrade BuildRegistry for v2 multi-directory scan

**Files:**
- Modify: `internal/asset/builder.go`
- Modify: `internal/asset/builder_test.go`

- [ ] **Step 1: Write failing tests**

```go
// Add to internal/asset/builder_test.go

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

	// global/theme/v1.0.0
	d = filepath.Join(dir, "global", "theme", "v1.0.0")
	os.MkdirAll(d, 0755)
	writeMetaGlobal(t, d, "theme", "1.0.0", "site/src/app/styles", []string{"globals.css"})
	os.WriteFile(filepath.Join(d, "globals.css"), []byte(":root{}"), 0644)

	reg, warnings, err := BuildRegistryV2(dir)
	if err != nil {
		t.Fatalf("BuildRegistryV2: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	// Check keys
	expected := []string{"scaffold/default", "ui/spinner", "components/data-table", "global/theme"}
	for _, key := range expected {
		if _, ok := reg[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}

	// Check paths
	if reg["ui/spinner"].Versions["1.0.0"].Path != "ui/spinner/v1.0.0" {
		t.Errorf("spinner path = %q", reg["ui/spinner"].Versions["1.0.0"].Path)
	}
}

func TestBuildRegistryV2_CSSWarning(t *testing.T) {
	dir := t.TempDir()

	// ui/bad-component with .css file
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

// Test helpers
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

func writeMetaGlobal(t *testing.T, dir, name, version, dest string, files []string) {
	t.Helper()
	m := Meta{Name: name, Version: version, Category: "global", Dest: dest, Files: files}
	data, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/asset/ -run "TestBuildRegistryV2" -v`
Expected: FAIL (BuildRegistryV2 doesn't exist)

- [ ] **Step 3: Implement BuildRegistryV2 and WriteRegistryV2**

Add to `internal/asset/builder.go`:

```go
var v2Categories = []string{"scaffold", "ui", "components", "global"}

// BuildRegistryV2 scans the four category directories and builds a v2 Registry.
// Returns the registry and any warnings (CSS in ui/components, category mismatch).
func BuildRegistryV2(rootDir string) (Registry, []string, error) {
	reg := make(Registry)
	var warnings []string

	for _, category := range v2Categories {
		catDir := filepath.Join(rootDir, category)
		entries, err := os.ReadDir(catDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, fmt.Errorf("read %s dir: %w", category, err)
		}

		expectedCategory := category
		if category == "components" {
			expectedCategory = "component"
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			assetName := entry.Name()
			registryKey := category + "/" + assetName

			versionDirs, err := os.ReadDir(filepath.Join(catDir, assetName))
			if err != nil {
				continue
			}

			info := AssetInfo{Versions: make(map[string]VersionEntry)}
			var versionStrings []string

			for _, vd := range versionDirs {
				if !vd.IsDir() || !strings.HasPrefix(vd.Name(), "v") {
					continue
				}

				versionDir := filepath.Join(catDir, assetName, vd.Name())
				metaPath := filepath.Join(versionDir, "meta.json")
				meta, err := LoadMeta(metaPath)
				if err != nil {
					continue
				}

				// Category mismatch warning
				if meta.Category != "" && meta.Category != expectedCategory {
					warnings = append(warnings, fmt.Sprintf(
						"Warning: %s meta.json category %q mismatches directory %q",
						registryKey, meta.Category, category))
				}

				// CSS file warning for ui/ and components/
				if category == "ui" || category == "components" {
					for _, f := range meta.Files {
						if strings.HasSuffix(f, ".css") {
							warnings = append(warnings, fmt.Sprintf(
								"Warning: %s contains CSS files. ui/ and components/ assets should use Tailwind classes only.", registryKey))
							break
						}
					}
					// Also check directory for undeclared CSS files
					dirEntries, _ := os.ReadDir(versionDir)
					for _, de := range dirEntries {
						if strings.HasSuffix(de.Name(), ".css") {
							warnings = append(warnings, fmt.Sprintf(
								"Warning: %s directory contains CSS file %s not declared in meta.json", registryKey, de.Name()))
							break
						}
					}
				}

				ver := meta.Version
				info.Versions[ver] = VersionEntry{
					Path: filepath.Join(category, assetName, vd.Name()),
				}
				versionStrings = append(versionStrings, ver)
			}

			if len(versionStrings) == 0 {
				continue
			}

			sorted := semver.SortVersions(versionStrings)
			info.Latest = sorted[len(sorted)-1]
			reg[registryKey] = info
		}
	}

	return reg, warnings, nil
}

// WriteRegistryV2 writes a v2 registry.json with $schema and version fields.
func WriteRegistryV2(reg Registry, path string) error {
	// Build ordered output: $schema, version, then assets
	output := make(map[string]interface{})
	output["$schema"] = "https://gve.dev/schema/registry.json"
	output["version"] = "2"
	for k, v := range reg {
		output[k] = v
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/asset/ -run "TestBuildRegistryV2" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/asset/builder.go internal/asset/builder_test.go
git commit -m "feat(builder): add BuildRegistryV2 with multi-dir scan and CSS validation"
```

---

### Task 2.2: Upgrade installer with category-based paths and peerDeps

**Files:**
- Modify: `internal/asset/installer.go`
- Modify: `internal/asset/installer_test.go`

- [ ] **Step 1: Write failing tests**

```go
// Add to internal/asset/installer_test.go

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
		{"", "button", "", "site/src/shared/wk/ui/button"},  // fallback
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
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/asset/ -run "TestInstallUIAsset_V2|TestGetInstallPath" -v`
Expected: FAIL

- [ ] **Step 3: Implement category-aware installer**

Update `internal/asset/installer.go`:

The key changes:
1. Add `GetInstallPath(category, name, dest) string` exported function
2. In `InstallUIAsset`, after loading meta, derive category from `meta.Category` or infer from `assetPath`, then use `GetInstallPath` instead of hardcoded `shared/ui/`
3. Extract the bare asset name from category-prefixed names (e.g. `"ui/spinner"` -> `"spinner"`)

```go
// GetInstallPath returns the project-relative install directory for an asset.
func GetInstallPath(category, name, dest string) string {
	if dest != "" {
		return dest
	}
	switch category {
	case "ui":
		return fmt.Sprintf("site/src/shared/wk/ui/%s", name)
	case "component":
		return fmt.Sprintf("site/src/shared/wk/components/%s", name)
	case "scaffold":
		return "site"
	case "global":
		return "" // global must have dest
	default:
		// Backward compat fallback
		return fmt.Sprintf("site/src/shared/wk/ui/%s", name)
	}
}
```

Update `InstallUIAsset` to use category routing:

```go
func InstallUIAsset(mgr *Manager, assetName, versionConstraint, projectDir string) (installedVersion string, err error) {
	reg, err := mgr.GetRegistry("ui")
	if err != nil {
		return "", fmt.Errorf("load registry: %w", err)
	}

	// Resolve version
	var assetPath string
	if versionConstraint == "" || versionConstraint == "latest" {
		ver, p, err := reg.GetLatest(assetName)
		if err != nil {
			return "", err
		}
		installedVersion = ver
		assetPath = p
	} else {
		p, err := reg.GetVersion(assetName, versionConstraint)
		if err != nil {
			return "", err
		}
		assetPath = p
		meta, err := LoadMeta(filepath.Join(mgr.GetAssetDir("ui", assetPath), "meta.json"))
		if err != nil {
			return "", fmt.Errorf("load meta: %w", err)
		}
		installedVersion = meta.Version
	}

	// Load meta
	srcDir := mgr.GetAssetDir("ui", assetPath)
	meta, err := LoadMeta(filepath.Join(srcDir, "meta.json"))
	if err != nil {
		return "", fmt.Errorf("load meta: %w", err)
	}

	// Determine category
	category := meta.Category
	if category == "" {
		category = InferCategory(assetPath)
	}

	// Determine destination
	bareName := meta.Name // Use the name from meta (bare, no prefix)
	installPath := GetInstallPath(category, bareName, meta.Dest)
	destDir := filepath.Join(projectDir, installPath)

	// Copy files
	if err := CopyAsset(srcDir, destDir, meta.Files); err != nil {
		return "", fmt.Errorf("copy asset: %w", err)
	}

	// Inject npm deps into package.json
	if len(meta.Deps) > 0 {
		pkgPath := filepath.Join(projectDir, "site", "package.json")
		if err := injectDeps(pkgPath, meta.Deps); err != nil {
			return "", fmt.Errorf("inject deps: %w", err)
		}
	}

	return installedVersion, nil
}
```

- [ ] **Step 4: Run installer tests**

Run: `go test ./internal/asset/ -run "TestInstallUIAsset|TestGetInstallPath" -v`
Expected: PASS

- [ ] **Step 5: Update existing installer test for v1-compat**

The existing `TestInstallUIAsset` uses v1 registry keys (`"button"`, path `"assets/button/v1.0.0"`). These still work since there's no category prefix, so `InferCategory("assets/button/v1.0.0")` returns `""` and fallback goes to `shared/wk/ui/button`. But the old test expects `shared/ui/button`. Update the old test to expect `shared/wk/ui/button`.

- [ ] **Step 6: Run ALL tests**

Run: `go test ./... -count=1`
Expected: PASS (after updating all path expectations)

- [ ] **Step 7: Commit**

```bash
git add internal/asset/installer.go internal/asset/installer_test.go
git commit -m "feat(installer): category-based install paths (shared/wk/ui/, shared/wk/components/)"
```

---

### Task 2.3: Add peerDeps resolution to installer

**Files:**
- Modify: `internal/asset/installer.go`
- Modify: `internal/asset/installer_test.go`

- [ ] **Step 1: Write failing test for peerDeps**

```go
func TestInstallUIAsset_PeerDeps(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	projectDir := filepath.Join(tmp, "project")

	regDir := filepath.Join(cacheDir, "ui")
	os.MkdirAll(regDir, 0755)

	// ui/spinner
	spinnerDir := filepath.Join(regDir, "ui", "spinner", "v1.0.0")
	os.MkdirAll(spinnerDir, 0755)
	spinnerMeta := Meta{Name: "spinner", Version: "1.0.0", Category: "ui", Files: []string{"spinner.tsx"}}
	sm, _ := json.Marshal(spinnerMeta)
	os.WriteFile(filepath.Join(spinnerDir, "meta.json"), sm, 0644)
	os.WriteFile(filepath.Join(spinnerDir, "spinner.tsx"), []byte("export const Spinner = () => null"), 0644)

	// components/data-table with peerDep on ui/spinner
	dtDir := filepath.Join(regDir, "components", "data-table", "v1.0.0")
	os.MkdirAll(dtDir, 0755)
	dtMeta := Meta{Name: "data-table", Version: "1.0.0", Category: "component", PeerDeps: []string{"ui/spinner"}, Files: []string{"data-table.tsx"}}
	dm, _ := json.Marshal(dtMeta)
	os.WriteFile(filepath.Join(dtDir, "meta.json"), dm, 0644)
	os.WriteFile(filepath.Join(dtDir, "data-table.tsx"), []byte("export const DataTable = () => null"), 0644)

	reg := Registry{
		"ui/spinner": {Latest: "1.0.0", Versions: map[string]VersionEntry{
			"1.0.0": {Path: "ui/spinner/v1.0.0"},
		}},
		"components/data-table": {Latest: "1.0.0", Versions: map[string]VersionEntry{
			"1.0.0": {Path: "components/data-table/v1.0.0"},
		}},
	}
	regData, _ := json.MarshalIndent(reg, "", "  ")
	os.WriteFile(filepath.Join(regDir, "registry.json"), regData, 0644)

	siteDir := filepath.Join(projectDir, "site")
	os.MkdirAll(siteDir, 0755)
	os.WriteFile(filepath.Join(siteDir, "package.json"), []byte(`{"name":"app","dependencies":{}}`), 0644)

	mgr := NewManager(cacheDir)

	// Create a lock file with no assets installed
	lockAssets := make(map[string]bool) // tracks what's "installed"

	// ResolvePeerDeps should return ["ui/spinner"]
	meta := dtMeta
	missing := ResolvePeerDeps(&meta, lockAssets)
	if len(missing) != 1 || missing[0] != "ui/spinner" {
		t.Errorf("ResolvePeerDeps = %v, want [ui/spinner]", missing)
	}

	// After marking spinner as installed, should return empty
	lockAssets["ui/spinner"] = true
	missing = ResolvePeerDeps(&meta, lockAssets)
	if len(missing) != 0 {
		t.Errorf("ResolvePeerDeps after install = %v, want empty", missing)
	}

	_ = mgr // used in integration test below
}

func TestResolvePeerDeps_CircularDetection(t *testing.T) {
	// A depends on B, B depends on A
	metaA := Meta{PeerDeps: []string{"ui/b"}}
	metaB := Meta{PeerDeps: []string{"ui/a"}}

	installed := map[string]bool{}
	missing := ResolvePeerDeps(&metaA, installed)
	if len(missing) != 1 || missing[0] != "ui/b" {
		t.Errorf("expected [ui/b], got %v", missing)
	}

	// After installing B, resolving B's peerDeps should want ui/a
	installed["ui/b"] = true
	missing = ResolvePeerDeps(&metaB, installed)
	if len(missing) != 1 || missing[0] != "ui/a" {
		t.Errorf("expected [ui/a], got %v", missing)
	}

	_ = metaA
}
```

- [ ] **Step 2: Implement ResolvePeerDeps**

Add to `internal/asset/installer.go`:

```go
// ResolvePeerDeps returns the list of peerDeps not yet installed.
func ResolvePeerDeps(meta *Meta, installed map[string]bool) []string {
	if len(meta.PeerDeps) == 0 {
		return nil
	}
	var missing []string
	for _, dep := range meta.PeerDeps {
		if !installed[dep] {
			missing = append(missing, dep)
		}
	}
	return missing
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/asset/ -run "TestInstallUIAsset_PeerDeps|TestResolvePeerDeps" -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/asset/installer.go internal/asset/installer_test.go
git commit -m "feat(installer): add ResolvePeerDeps for v2 peerDeps resolution"
```

---

## Phase 3: CLI Commands

### Task 3.1: Update `gve registry build` for v2

**Files:**
- Modify: `internal/cmd/registry_build.go`
- Modify: `internal/cmd/user_journey_test.go` (TestRegistryBuild_MultiVersion)

- [ ] **Step 1: Update registry_build.go**

Replace `runRegistryBuild` to use `BuildRegistryV2` and `WriteRegistryV2`. Remove `--dir` flag, add `--root` flag defaulting to `.`.

```go
func newRegistryBuildCmd() *cobra.Command {
	var rootDir string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "扫描 scaffold/ui/components/global/ 目录，生成 registry.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegistryBuild(rootDir)
		},
	}

	cmd.Flags().StringVar(&rootDir, "root", ".", "仓库根目录路径")
	return cmd
}

func runRegistryBuild(rootDir string) error {
	absDir, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	fmt.Printf("Scanning %s ...\n", absDir)

	reg, warnings, err := asset.BuildRegistryV2(absDir)
	if err != nil {
		return fmt.Errorf("build registry: %w", err)
	}

	for _, w := range warnings {
		fmt.Printf("  %s\n", w)
	}

	outPath := filepath.Join(absDir, "registry.json")
	if err := asset.WriteRegistryV2(reg, outPath); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	// Count by category
	counts := map[string]int{}
	for key := range reg {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) == 2 {
			counts[parts[0]]++
		}
	}

	for _, cat := range []string{"scaffold", "ui", "components", "global"} {
		if c := counts[cat]; c > 0 {
			fmt.Printf("  %s/: %d assets\n", cat, c)
		}
	}

	fmt.Printf("\n✓ registry.json updated (%d assets, %d warnings)\n", len(reg), len(warnings))
	return nil
}
```

- [ ] **Step 2: Update TestRegistryBuild_MultiVersion for v2 structure**

Change the test to create `ui/widget/v{x}` directories instead of `assets/widget/v{x}`, and check for `"ui/widget"` key.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/cmd/ -run "TestRegistryBuild" -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/registry_build.go internal/cmd/user_journey_test.go
git commit -m "feat(cmd): update registry build for v2 multi-dir scan"
```

---

### Task 3.2: Update `gve ui add` with category prefix and peerDeps

**Files:**
- Modify: `internal/cmd/ui_add.go`

- [ ] **Step 1: Update runUIAdd**

Key changes:
1. Use `reg.ResolveAssetName(name)` to expand shortnames
2. After install, resolve peerDeps and auto-install missing ones
3. Use v2-style lock keys

```go
func runUIAdd(cmd *cobra.Command, args []string) error {
	name, version := parseAssetArg(args[0])

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	cfg := config.Default()
	mgr := asset.NewManager(cfg.CacheDir)

	fmt.Printf("Updating UI registry cache...\n")
	if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}

	reg, err := mgr.GetRegistry("ui")
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	// Resolve shortname to full key
	fullName, ok := reg.ResolveAssetName(name)
	if !ok {
		return fmt.Errorf("asset %q not found in registry. Try: gve ui add ui/%s or components/%s", name, name, name)
	}

	fmt.Printf("Installing %s", fullName)
	if version != "" {
		fmt.Printf("@%s", version)
	}
	fmt.Println("...")

	installedVer, err := asset.InstallUIAsset(mgr, fullName, version, projectDir)
	if err != nil {
		return fmt.Errorf("install %s: %w", fullName, err)
	}

	// Load lock and track installed assets
	lockPath := filepath.Join(projectDir, "gve.lock")
	lf, err := lock.Load(lockPath)
	if err != nil {
		return fmt.Errorf("load gve.lock: %w", err)
	}
	lf.SetUIAsset(fullName, installedVer)

	// Resolve peerDeps
	assetPath := ""
	if version == "" || version == "latest" {
		_, assetPath, _ = reg.GetLatest(fullName)
	} else {
		assetPath, _ = reg.GetVersion(fullName, version)
	}
	if assetPath != "" {
		meta, _ := asset.LoadMeta(filepath.Join(mgr.GetAssetDir("ui", assetPath), "meta.json"))
		if meta != nil && len(meta.PeerDeps) > 0 {
			installed := make(map[string]bool)
			for k := range lf.UI.Assets {
				installed[k] = true
			}
			installed[fullName] = true

			missing := asset.ResolvePeerDeps(meta, installed)
			for _, dep := range missing {
				fmt.Printf("  → peerDep %s: installing...\n", dep)
				depVer, err := asset.InstallUIAsset(mgr, dep, "", projectDir)
				if err != nil {
					fmt.Printf("  → peerDep %s: failed: %v\n", dep, err)
					continue
				}
				lf.SetUIAsset(dep, depVer)
				fmt.Printf("  → peerDep %s: installed v%s\n", dep, depVer)
			}
		}
	}

	if err := lf.Save(lockPath); err != nil {
		return fmt.Errorf("save gve.lock: %w", err)
	}

	fmt.Printf("✓ Installed %s@%s\n", fullName, installedVer)
	return nil
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/ui_add.go
git commit -m "feat(cmd): ui add supports category prefixes, shortnames, and peerDeps"
```

---

### Task 3.3: Update `gve ui list` with category grouping

**Files:**
- Modify: `internal/cmd/ui_list.go`

- [ ] **Step 1: Implement category-grouped list**

Replace `runUIList` to group assets by category prefix and show v2 paths.

```go
func runUIList(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	lockPath := filepath.Join(projectDir, "gve.lock")
	lf, err := lock.Load(lockPath)
	if err != nil {
		return fmt.Errorf("load gve.lock: %w", err)
	}

	if len(lf.UI.Assets) == 0 {
		fmt.Println("No UI assets installed.")
		return nil
	}

	cfg := config.Default()
	mgr := asset.NewManager(cfg.CacheDir)
	reg, _ := mgr.GetRegistry("ui")

	// Group by category
	categories := []struct {
		label  string
		prefix string
	}{
		{"SCAFFOLD", "scaffold/"},
		{"UI", "ui/"},
		{"COMPONENTS", "components/"},
		{"GLOBAL", "global/"},
	}

	printed := false
	for _, cat := range categories {
		var names []string
		for name := range lf.UI.Assets {
			if strings.HasPrefix(name, cat.prefix) {
				names = append(names, name)
			}
		}
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)

		if printed {
			fmt.Println()
		}
		fmt.Println(cat.label)
		for _, name := range names {
			entry := lf.UI.Assets[name]
			dest := resolveDestPath(name, entry.Version, reg, mgr)
			fmt.Printf("  %-30s v%-10s %s\n", name, entry.Version, dest)
		}
		printed = true
	}

	// Print any uncategorized assets (v1 compat)
	var uncategorized []string
	for name := range lf.UI.Assets {
		hasPrefix := false
		for _, cat := range categories {
			if strings.HasPrefix(name, cat.prefix) {
				hasPrefix = true
				break
			}
		}
		if !hasPrefix {
			uncategorized = append(uncategorized, name)
		}
	}
	if len(uncategorized) > 0 {
		sort.Strings(uncategorized)
		if printed {
			fmt.Println()
		}
		fmt.Println("OTHER")
		for _, name := range uncategorized {
			entry := lf.UI.Assets[name]
			dest := resolveDestPath(name, entry.Version, reg, mgr)
			fmt.Printf("  %-30s v%-10s %s\n", name, entry.Version, dest)
		}
	}

	return nil
}
```

Update `resolveDestPath` to use v2 category-aware paths:

```go
func resolveDestPath(name, version string, reg asset.Registry, mgr *asset.Manager) string {
	if reg == nil {
		return ""
	}

	info, ok := reg[name]
	if !ok {
		// Fallback: derive from name pattern
		category := asset.InferCategory(name)
		bareName := name
		if idx := strings.Index(name, "/"); idx >= 0 {
			bareName = name[idx+1:]
		}
		path := asset.GetInstallPath(category, bareName, "")
		if path != "" {
			return path + "/"
		}
		return ""
	}

	ve, ok := info.Versions[version]
	if !ok {
		return ""
	}

	metaPath := filepath.Join(mgr.GetAssetDir("ui", ve.Path), "meta.json")
	meta, err := asset.LoadMeta(metaPath)
	if err != nil {
		return ""
	}

	category := meta.Category
	if category == "" {
		category = asset.InferCategory(ve.Path)
	}
	path := asset.GetInstallPath(category, meta.Name, meta.Dest)
	if path != "" {
		return path + "/"
	}
	return ""
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/ui_list.go
git commit -m "feat(cmd): ui list groups assets by category"
```

---

### Task 3.4: Update `gve ui diff` with v2 paths and interactive actions

**Files:**
- Modify: `internal/cmd/ui_diff.go`

- [ ] **Step 1: Update runUIDiff to use v2 path resolution and add u/k/m/s actions**

Key changes:
1. Use `GetInstallPath` + `InferCategory` for local dir resolution
2. After showing diff, prompt for action: upgrade/keep/merge/skip
3. Implement each action

- [ ] **Step 2: Run tests**

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/ui_diff.go
git commit -m "feat(cmd): ui diff supports v2 paths and upgrade/keep/merge/skip actions"
```

---

### Task 3.5: Update `gve ui sync` with v2 paths and 4-option flow

**Files:**
- Modify: `internal/cmd/ui_sync.go`

- [ ] **Step 1: Update runUISync**

Key changes:
1. Show preflight summary of all assets with versions and local-changes annotation
2. Use v2 path resolution
3. Change prompt to u/k/m/s (upgrade/keep/merge/skip)

- [ ] **Step 2: Run tests**

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/ui_sync.go
git commit -m "feat(cmd): ui sync with preflight summary and 4-option interactive flow"
```

---

### Task 3.6: Update `gve init` for scaffold selection

**Files:**
- Modify: `internal/cmd/init_frontend.go`
- Modify: `internal/cmd/init.go` (update output text)

- [ ] **Step 1: Update initFrontend to select scaffold from registry**

Key changes:
1. Filter `reg.ListByCategory("scaffold")`
2. If one scaffold: auto-select; if multiple: prompt
3. Use `scaffold/default` as lock key
4. Create v2 placeholder dirs (`shared/wk/ui/`, `shared/wk/components/`)

- [ ] **Step 2: Run tests**

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/init_frontend.go internal/cmd/init.go
git commit -m "feat(cmd): init selects scaffold from registry, v2 placeholder paths"
```

---

### Task 3.7: Update `gve sync` and `gve status` for v2

**Files:**
- Modify: `internal/cmd/sync.go`
- Modify: `internal/cmd/status.go`

- [ ] **Step 1: Update assetExists in sync.go to use v2 paths**

Replace hardcoded `shared/ui/` with `GetInstallPath` + `InferCategory`.

- [ ] **Step 2: Run ALL tests**

Run: `go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/sync.go internal/cmd/status.go
git commit -m "feat(cmd): sync and status use v2 category-aware paths"
```

---

### Task 3.8: Update ALL integration tests for v2

**Files:**
- Modify: `internal/cmd/user_journey_test.go`

- [ ] **Step 1: Full overhaul of setupFakeAssetCache to v2 structure**

Replace `assets/` paths with `scaffold/`, `ui/` paths. Use v2 registry keys. Use v2 meta with category field. Create registry.json with v2 format (version, $schema).

- [ ] **Step 2: Update all test step expectations**

- `Step1_UIAdd_Button`: install path `shared/wk/ui/button/`, lock key `ui/button`
- `Step5_UIDiff_NoChanges`: local path `shared/wk/ui/button/`
- `Step6_UIDiff_WithLocalEdit`: local path `shared/wk/ui/button/`
- `Step7_Sync_RestoresMissing`: path `shared/wk/ui/button/`
- `Step9_UIUpgrade_ToLatest`: path `shared/wk/ui/button/`
- `TestInstallGlobalAsset`: use `scaffold/default` key
- `TestResolveDestPath`: v2 paths
- `TestPrintAssetStatus`: v2 keys
- `TestAssetExists_*`: v2 keys and paths

- [ ] **Step 3: Run ALL tests**

Run: `go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/user_journey_test.go
git commit -m "test: update all integration tests for v2 asset architecture"
```

---

## Phase 4: wk-ui Remote Repository Rebuild

### Task 4.1: Clone wk-ui, rebuild with v2 structure, force push

**Files:**
- External: `castle-x/wk-ui` GitHub repository

- [ ] **Step 1: Clone wk-ui**

```bash
cd /tmp && git clone git@github.com:castle-x/wk-ui.git wk-ui-v2 && cd wk-ui-v2
```

- [ ] **Step 2: Remove old structure**

```bash
rm -rf assets/ registry.json
```

- [ ] **Step 3: Create v2 directory structure with minimal assets**

Create `scaffold/default/v1.0.0/` — copy existing base-setup content, update meta.json to v2 format.

Create `ui/spinner/v1.0.0/` — minimal spinner component with v2 meta.json.

Create `components/data-table/v1.0.0/` — minimal data-table with peerDeps on ui/spinner, v2 meta.json.

Create `global/theme/v1.0.0/` — globals.css with v2 meta.json.

- [ ] **Step 4: Generate registry.json using gve**

Build gve first (`make build`), then run:

```bash
./gve registry build --root /tmp/wk-ui-v2
```

Verify `registry.json` has v2 format with `$schema`, `version: "2"`, category-prefixed keys.

- [ ] **Step 5: Commit and force push**

```bash
cd /tmp/wk-ui-v2
git add -A
git commit -m "feat: rebuild wk-ui with v2 category structure (scaffold/ui/components/global)"
git push --force origin main
```

- [ ] **Step 6: Verify remote**

```bash
gh api repos/castle-x/wk-ui/contents/ --jq '.[].name'
# Should show: components, global, registry.json, scaffold, ui
```

---

## Phase 5: End-to-End Validation

### Task 5.1: Full functional test with real wk-ui

- [ ] **Step 1: Build and install gve v2**

```bash
cd /data/home/castlexu/github/gve
make build && make install
```

- [ ] **Step 2: Test `gve init`**

```bash
cd /tmp && rm -rf test-v2-project
gve init test-v2-project
# Verify: scaffold selected, site/ created, gve.lock has scaffold/default
cat test-v2-project/gve.lock
```

- [ ] **Step 3: Test `gve ui add` with full name**

```bash
cd /tmp/test-v2-project
gve ui add ui/spinner
# Verify: installed to site/src/shared/wk/ui/spinner/
ls site/src/shared/wk/ui/spinner/
cat gve.lock
```

- [ ] **Step 4: Test `gve ui add` with shortname**

```bash
gve ui add data-table
# Verify: resolves to components/data-table, installs peerDeps
ls site/src/shared/wk/components/data-table/
cat gve.lock
```

- [ ] **Step 5: Test `gve ui list`**

```bash
gve ui list
# Verify: grouped by SCAFFOLD, UI, COMPONENTS, GLOBAL
```

- [ ] **Step 6: Test `gve status`**

```bash
gve status
```

- [ ] **Step 7: Test `gve ui diff`**

```bash
# Modify a file
echo "// custom" >> site/src/shared/wk/ui/spinner/spinner.tsx
gve ui diff ui/spinner
```

- [ ] **Step 8: Test `gve registry build` (in wk-ui clone)**

```bash
cd ~/.gve/cache/ui
gve registry build
cat registry.json | head -20
```

- [ ] **Step 9: Run all unit tests one final time**

```bash
cd /data/home/castlexu/github/gve
go test ./... -count=1 -v
```

- [ ] **Step 10: Tag v2 release**

```bash
git tag v0.2.0
git push origin v0.2.0
```

---

## Summary: Phase Dependencies

```
Phase 1 (data layer)
  └─ Task 1.1: meta.go (no deps)
  └─ Task 1.2: lock.go (no deps)
  └─ Task 1.3: registry.go + fix integration tests (depends on 1.1, 1.2)

Phase 2 (asset ops) — depends on Phase 1
  └─ Task 2.1: builder.go (depends on 1.1)
  └─ Task 2.2: installer.go (depends on 1.1, 1.3)
  └─ Task 2.3: peerDeps (depends on 2.2)

Phase 3 (CLI commands) — depends on Phase 2
  └─ Task 3.1: registry build cmd (depends on 2.1)
  └─ Task 3.2: ui add cmd (depends on 2.2, 2.3)
  └─ Task 3.3: ui list cmd (depends on 2.2)
  └─ Task 3.4: ui diff cmd (depends on 2.2)
  └─ Task 3.5: ui sync cmd (depends on 2.2)
  └─ Task 3.6: init cmd (depends on 2.2)
  └─ Task 3.7: sync + status cmd (depends on 2.2)
  └─ Task 3.8: integration test overhaul (depends on 3.1-3.7)

Phase 4 (wk-ui rebuild) — depends on Phase 3 (needs working gve registry build)
  └─ Task 4.1: clone, rebuild, push

Phase 5 (E2E validation) — depends on Phase 3 + 4
  └─ Task 5.1: full functional test + tag release
```
