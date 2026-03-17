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
