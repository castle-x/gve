package asset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// InstallUIAsset installs a UI asset from the cache into the project directory.
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
		// Extract the actual resolved version from meta
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

	// Determine destination
	var destDir string
	if meta.Dest != "" {
		destDir = filepath.Join(projectDir, meta.Dest)
	} else {
		destDir = filepath.Join(projectDir, "site", "src", "shared", "ui", assetName)
	}

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

// injectDeps adds npm dependencies to package.json if not already present.
func injectDeps(pkgPath string, deps []string) error {
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("parse package.json: %w", err)
	}

	depsMap, ok := pkg["dependencies"].(map[string]interface{})
	if !ok {
		depsMap = make(map[string]interface{})
		pkg["dependencies"] = depsMap
	}

	changed := false
	for _, dep := range deps {
		if _, exists := depsMap[dep]; !exists {
			depsMap[dep] = "latest"
			changed = true
		}
	}

	if !changed {
		return nil
	}

	out, err := marshalJSONOrdered(pkg)
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}

	return os.WriteFile(pkgPath, out, 0644)
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
