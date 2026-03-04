package asset

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallAPIAsset installs an API contract from the cache into the project directory.
// resourcePath is in the form "project/resource" (e.g. "ai-worker/task").
// API versions are major-only (v1, v2), not semver.
func InstallAPIAsset(mgr *Manager, resourcePath, version, projectDir string) (string, error) {
	reg, err := mgr.GetRegistry("api")
	if err != nil {
		return "", fmt.Errorf("load api registry: %w", err)
	}

	var assetPath string
	var installedVersion string

	if version == "" || version == "latest" {
		ver, p, err := reg.GetLatest(resourcePath)
		if err != nil {
			return "", err
		}
		installedVersion = ver
		assetPath = p
	} else {
		info, ok := reg[resourcePath]
		if !ok {
			return "", fmt.Errorf("API %q not found in registry", resourcePath)
		}
		entry, ok := info.Versions[version]
		if !ok {
			return "", fmt.Errorf("version %s of API %q not found", version, resourcePath)
		}
		installedVersion = version
		assetPath = entry.Path
	}

	srcDir := mgr.GetAssetDir("api", assetPath)

	destDir := filepath.Join(projectDir, "api", resourcePath, installedVersion)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return "", fmt.Errorf("read api asset dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".thrift" {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no thrift file found in API asset %q@%s", resourcePath, installedVersion)
	}

	if err := CopyAsset(srcDir, destDir, files); err != nil {
		return "", fmt.Errorf("copy api asset: %w", err)
	}

	return installedVersion, nil
}

// APIAssetDirExists checks if an API asset version directory already exists in the project.
func APIAssetDirExists(projectDir, resourcePath, version string) bool {
	dir := filepath.Join(projectDir, "api", resourcePath, version)
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return info.IsDir()
}
