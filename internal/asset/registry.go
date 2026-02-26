package asset

import (
	"encoding/json"
	"fmt"
	"os"

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

func LoadRegistry(path string) (Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
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
	return names
}
