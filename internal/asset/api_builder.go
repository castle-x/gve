package asset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// vNPattern matches API version directories like "v1", "v2", "v10".
var vNPattern = regexp.MustCompile(`^v(\d+)$`)

// BuildAPIRegistry scans an API cache directory and builds a registry.
// Directory structure: {project}/{resource}/{vN}/*.thrift
func BuildAPIRegistry(rootDir string) (Registry, error) {
	reg := make(Registry)

	// First level: project directories
	projects, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("read api root dir: %w", err)
	}

	for _, proj := range projects {
		if !proj.IsDir() {
			continue
		}
		projName := proj.Name()

		// Skip hidden dirs and special files
		if strings.HasPrefix(projName, ".") {
			continue
		}

		// Second level: resource directories
		projDir := filepath.Join(rootDir, projName)
		resources, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}

		for _, res := range resources {
			if !res.IsDir() {
				continue
			}
			resName := res.Name()
			registryKey := projName + "/" + resName

			// Third level: version directories (vN)
			resDir := filepath.Join(projDir, resName)
			versionDirs, err := os.ReadDir(resDir)
			if err != nil {
				continue
			}

			info := AssetInfo{Versions: make(map[string]VersionEntry)}
			var majorVersions []int

			for _, vd := range versionDirs {
				if !vd.IsDir() {
					continue
				}

				matches := vNPattern.FindStringSubmatch(vd.Name())
				if matches == nil {
					continue
				}

				// Verify at least one .thrift file exists
				vDir := filepath.Join(resDir, vd.Name())
				if !hasThriftFiles(vDir) {
					continue
				}

				major, _ := strconv.Atoi(matches[1])
				majorVersions = append(majorVersions, major)
				info.Versions[vd.Name()] = VersionEntry{
					Path: filepath.Join(projName, resName, vd.Name()),
				}
			}

			if len(majorVersions) == 0 {
				continue
			}

			// Latest = highest major version
			sort.Ints(majorVersions)
			info.Latest = fmt.Sprintf("v%d", majorVersions[len(majorVersions)-1])
			reg[registryKey] = info
		}
	}

	return reg, nil
}

// WriteAPIRegistry writes an API registry.json.
func WriteAPIRegistry(reg Registry, path string) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal api registry: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// hasThriftFiles checks if a directory contains at least one .thrift file.
func hasThriftFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".thrift" {
			return true
		}
	}
	return false
}
