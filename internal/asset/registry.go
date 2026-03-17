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
	// Exact match (already has category prefix or v1 bare name)
	if _, ok := r[name]; ok {
		return name, true
	}

	// Try shortname resolution with priority order
	for _, prefix := range []string{"ui/", "components/", "global/"} {
		fullName := prefix + name
		if _, ok := r[fullName]; ok {
			return fullName, true
		}
	}

	return "", false
}

// ListByCategory returns all asset names matching the given category prefix, sorted.
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
