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
