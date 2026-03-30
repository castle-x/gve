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

var v2Categories = []string{"scaffold", "ui", "components", "hooks"}

// BuildRegistryV2 scans the four category directories and builds a v2 Registry.
// Returns the registry and any warnings (CSS in ui/components, category mismatch).
func BuildRegistryV2(rootDir string) (Registry, []string, error) {
	return BuildRegistryV2Filtered(rootDir, v2Categories)
}

// BuildRegistryV2Filtered scans only the specified category directories and builds a v2 Registry.
// If the rootDir contains an existing registry.json, entries for non-selected categories are preserved.
func BuildRegistryV2Filtered(rootDir string, categories []string) (Registry, []string, error) {
	reg := make(Registry)

	// Load existing registry to preserve entries for non-selected categories
	existingPath := filepath.Join(rootDir, "registry.json")
	if existing, err := LoadRegistry(existingPath); err == nil {
		// Copy entries whose category is NOT in the selected set
		selected := make(map[string]bool, len(categories))
		for _, c := range categories {
			selected[c] = true
		}
		for key, info := range existing {
			parts := strings.SplitN(key, "/", 2)
			if len(parts) == 2 && !selected[parts[0]] {
				reg[key] = info
			}
		}
	}

	var warnings []string

	for _, category := range categories {
		catDir := filepath.Join(rootDir, category)
		entries, err := os.ReadDir(catDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, fmt.Errorf("read %s dir: %w", category, err)
		}

		expectedCategory := category
		switch category {
		case "components":
			expectedCategory = "component"
		case "hooks":
			expectedCategory = "hook"
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
				if category == "ui" || category == "components" || category == "hooks" {
					for _, f := range meta.Files {
						if strings.HasSuffix(f, ".css") {
							warnings = append(warnings, fmt.Sprintf(
								"Warning: %s contains CSS files. ui/ and components/ assets should use Tailwind classes only.", registryKey))
							break
						}
					}

					// Scan for undeclared CSS files in directory
					declared := make(map[string]bool)
					for _, f := range meta.Files {
						declared[f] = true
					}
					if actualFiles, err := collectAllFiles(versionDir); err == nil {
						for _, f := range actualFiles {
							if strings.HasSuffix(f, ".css") && !declared[f] {
								warnings = append(warnings, fmt.Sprintf(
									"Warning: %s has undeclared CSS file %q in directory", registryKey, f))
							}
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

// collectAllFiles recursively lists all files in dir, returning paths relative to dir.
func collectAllFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "meta.json" {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}
