package asset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ResolvePeerDeps returns the list of peerDeps from meta that are not yet installed.
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

// ResolvePeerDepsRecursive recursively resolves peerDeps, returning a deduplicated
// list in topological order (leaf dependencies first). Uses BFS and detects cycles.
// maxDepth limits recursion depth (recommended: 5).
func ResolvePeerDepsRecursive(mgr *Manager, rootAsset string, installed map[string]bool, maxDepth int) ([]string, error) {
	reg, err := mgr.GetRegistry("ui")
	if err != nil {
		return nil, fmt.Errorf("load registry: %w", err)
	}

	type queueItem struct {
		name  string
		depth int
	}

	visited := make(map[string]bool)
	visited[rootAsset] = true
	for k := range installed {
		visited[k] = true
	}

	var result []string
	queue := []queueItem{}

	// Load root's peerDeps
	rootMeta := loadMetaForAsset(mgr, reg, rootAsset)
	if rootMeta == nil || len(rootMeta.PeerDeps) == 0 {
		return nil, nil
	}

	for _, dep := range rootMeta.PeerDeps {
		if !visited[dep] {
			visited[dep] = true
			queue = append(queue, queueItem{name: dep, depth: 1})
		}
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		result = append(result, item.name)

		if item.depth >= maxDepth {
			fmt.Printf("  \u26a0 peerDep chain depth limit (%d) reached at %s, stopping\n", maxDepth, item.name)
			continue
		}

		meta := loadMetaForAsset(mgr, reg, item.name)
		if meta == nil {
			continue
		}

		for _, dep := range meta.PeerDeps {
			if !visited[dep] {
				visited[dep] = true
				queue = append(queue, queueItem{name: dep, depth: item.depth + 1})
			}
		}
	}

	// Reverse for topological order (leaf dependencies first)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

// loadMetaForAsset loads the meta.json for the latest version of an asset from the registry cache.
func loadMetaForAsset(mgr *Manager, reg Registry, assetName string) *Meta {
	_, assetPath, err := reg.GetLatest(assetName)
	if err != nil {
		return nil
	}
	meta, err := LoadMeta(filepath.Join(mgr.GetAssetDir("ui", assetPath), "meta.json"))
	if err != nil {
		return nil
	}
	return meta
}

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

// InstallResult contains the result of an asset installation.
type InstallResult struct {
	InstalledVersion string
	DepsInjected     bool
}

// InstallUIAsset installs a UI asset from the cache into the project directory.
func InstallUIAsset(mgr *Manager, assetName, versionConstraint, projectDir string) (installedVersion string, err error) {
	result, err := InstallUIAssetFull(mgr, assetName, versionConstraint, projectDir)
	if err != nil {
		return "", err
	}
	return result.InstalledVersion, nil
}

// InstallUIAssetFull installs a UI asset and returns detailed result including whether deps were injected.
func InstallUIAssetFull(mgr *Manager, assetName, versionConstraint, projectDir string) (*InstallResult, error) {
	reg, err := mgr.GetRegistry("ui")
	if err != nil {
		return nil, fmt.Errorf("load registry: %w", err)
	}

	// Resolve version
	var assetPath string
	var installedVersion string
	if versionConstraint == "" || versionConstraint == "latest" {
		ver, p, err := reg.GetLatest(assetName)
		if err != nil {
			return nil, err
		}
		installedVersion = ver
		assetPath = p
	} else {
		p, err := reg.GetVersion(assetName, versionConstraint)
		if err != nil {
			return nil, err
		}
		assetPath = p
		// Extract the actual resolved version from meta
		meta, err := LoadMeta(filepath.Join(mgr.GetAssetDir("ui", assetPath), "meta.json"))
		if err != nil {
			return nil, fmt.Errorf("load meta: %w", err)
		}
		installedVersion = meta.Version
	}

	// Load meta
	srcDir := mgr.GetAssetDir("ui", assetPath)
	meta, err := LoadMeta(filepath.Join(srcDir, "meta.json"))
	if err != nil {
		return nil, fmt.Errorf("load meta: %w", err)
	}

	// Determine category
	category := meta.Category
	if category == "" {
		category = InferCategory(assetPath)
	}

	// Determine destination using category-aware path
	bareName := meta.Name
	installPath := GetInstallPath(category, bareName, meta.Dest)
	destDir := filepath.Join(projectDir, installPath)

	// Copy files
	if err := CopyAsset(srcDir, destDir, meta.Files); err != nil {
		return nil, fmt.Errorf("copy asset: %w", err)
	}

	// Inject npm deps into package.json
	depsInjected := false
	if len(meta.Deps) > 0 {
		pkgPath := filepath.Join(projectDir, "site", "package.json")
		changed, err := injectDeps(pkgPath, meta.Deps)
		if err != nil {
			return nil, fmt.Errorf("inject deps: %w", err)
		}
		depsInjected = changed
	}

	return &InstallResult{
		InstalledVersion: installedVersion,
		DepsInjected:     depsInjected,
	}, nil
}

// parseDep splits a dependency string into name and version.
// "lucide-react@^0.300.0" → ("lucide-react", "^0.300.0")
// "sonner" → ("sonner", "latest")
func parseDep(dep string) (name, version string) {
	if idx := strings.LastIndex(dep, "@"); idx > 0 {
		return dep[:idx], dep[idx+1:]
	}
	return dep, "latest"
}

// injectDeps adds npm dependencies to package.json if not already present.
// Returns true if any new dependencies were added.
func injectDeps(pkgPath string, deps []string) (bool, error) {
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return false, fmt.Errorf("read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false, fmt.Errorf("parse package.json: %w", err)
	}

	depsMap, ok := pkg["dependencies"].(map[string]interface{})
	if !ok {
		depsMap = make(map[string]interface{})
		pkg["dependencies"] = depsMap
	}

	changed := false
	for _, dep := range deps {
		depName, depVersion := parseDep(dep)
		if _, exists := depsMap[depName]; !exists {
			depsMap[depName] = depVersion
			changed = true
		}
	}

	if !changed {
		return false, nil
	}

	out, err := marshalJSONOrdered(pkg)
	if err != nil {
		return false, fmt.Errorf("marshal package.json: %w", err)
	}

	return true, os.WriteFile(pkgPath, out, 0644)
}

// marshalJSONOrdered produces deterministic JSON with sorted keys.
func marshalJSONOrdered(v map[string]interface{}) ([]byte, error) {
	// Sort top-level keys in a predictable order for package.json
	priority := map[string]int{
		"name": 0, "private": 1, "type": 2, "version": 3,
		"scripts": 4, "dependencies": 5, "devDependencies": 6,
	}

	type kv struct {
		key string
		val interface{}
	}

	var pairs []kv
	for k, val := range v {
		pairs = append(pairs, kv{k, val})
	}
	sort.Slice(pairs, func(i, j int) bool {
		pi, oki := priority[pairs[i].key]
		pj, okj := priority[pairs[j].key]
		if oki && okj {
			return pi < pj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return pairs[i].key < pairs[j].key
	})

	// Rebuild as ordered map via json.Marshal on a slice-based structure
	// Since Go maps don't guarantee order, we use json.Encoder manually
	buf := []byte("{\n")
	for idx, p := range pairs {
		keyJSON, _ := json.Marshal(p.key)
		valJSON, _ := json.MarshalIndent(p.val, "  ", "  ")
		buf = append(buf, []byte("  ")...)
		buf = append(buf, keyJSON...)
		buf = append(buf, []byte(": ")...)
		buf = append(buf, valJSON...)
		if idx < len(pairs)-1 {
			buf = append(buf, ',')
		}
		buf = append(buf, '\n')
	}
	buf = append(buf, []byte("}\n")...)
	return buf, nil
}
