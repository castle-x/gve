package asset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/semver"
)

// BuildRegistry scans assetsDir for {name}/v{x.y.z}/meta.json entries
// and builds a Registry (registry.json content).
func BuildRegistry(assetsDir string) (Registry, error) {
	reg := make(Registry)

	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		return nil, fmt.Errorf("read assets dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		assetName := entry.Name()

		versionDirs, err := os.ReadDir(filepath.Join(assetsDir, assetName))
		if err != nil {
			continue
		}

		info := AssetInfo{
			Versions: make(map[string]VersionEntry),
		}

		var versionStrings []string

		for _, vd := range versionDirs {
			if !vd.IsDir() || !strings.HasPrefix(vd.Name(), "v") {
				continue
			}

			metaPath := filepath.Join(assetsDir, assetName, vd.Name(), "meta.json")
			meta, err := LoadMeta(metaPath)
			if err != nil {
				continue
			}

			ver := meta.Version
			info.Versions[ver] = VersionEntry{
				Path: filepath.Join("assets", assetName, vd.Name()),
			}
			versionStrings = append(versionStrings, ver)
		}

		if len(versionStrings) == 0 {
			continue
		}

		sorted := semver.SortVersions(versionStrings)
		info.Latest = sorted[len(sorted)-1]
		reg[assetName] = info
	}

	return reg, nil
}

// WriteRegistry serializes a Registry to JSON and writes it to path.
func WriteRegistry(reg Registry, path string) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
