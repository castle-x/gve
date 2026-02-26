package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/lock"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "根据 gve.lock 还原所有资产",
		RunE:  runSync,
	}
}

func runSync(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	cfg := config.Default()
	lockPath := filepath.Join(projectDir, "gve.lock")
	lf, err := lock.Load(lockPath)
	if err != nil {
		return fmt.Errorf("load gve.lock: %w", err)
	}

	var installed, skipped int

	// Sync UI assets
	if len(lf.UI.Assets) > 0 {
		mgr := asset.NewManager(cfg.CacheDir)
		fmt.Println("Syncing UI assets...")
		if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
			return fmt.Errorf("update ui cache: %w", err)
		}

		reg, err := mgr.GetRegistry("ui")
		if err != nil {
			return fmt.Errorf("load ui registry: %w", err)
		}

		for name, entry := range lf.UI.Assets {
			if assetExists(name, entry.Version, reg, mgr, projectDir) {
				skipped++
				continue
			}

			fmt.Printf("  Installing %s@%s...\n", name, entry.Version)
			_, err := asset.InstallUIAsset(mgr, name, entry.Version, projectDir)
			if err != nil {
				fmt.Printf("  ✗ Failed to install %s: %v\n", name, err)
				continue
			}
			installed++
		}
	}

	// Sync API assets (placeholder for future)
	if len(lf.API.Assets) > 0 {
		fmt.Println("Syncing API assets...")
		for name := range lf.API.Assets {
			apiDir := filepath.Join(projectDir, "api", name)
			if _, err := os.Stat(apiDir); err == nil {
				skipped++
			} else {
				fmt.Printf("  ⚠ %s: API sync not yet implemented\n", name)
			}
		}
	}

	fmt.Printf("\nSync complete: %d installed, %d skipped (already present)\n", installed, skipped)
	return nil
}

// assetExists checks if an asset's files are already present in the project.
func assetExists(name, version string, reg asset.Registry, mgr *asset.Manager, projectDir string) bool {
	info, ok := reg[name]
	if !ok {
		return false
	}

	ve, ok := info.Versions[version]
	if !ok {
		return false
	}

	metaPath := filepath.Join(mgr.GetAssetDir("ui", ve.Path), "meta.json")
	meta, err := asset.LoadMeta(metaPath)
	if err != nil {
		return false
	}

	var destDir string
	if meta.Dest != "" {
		destDir = filepath.Join(projectDir, meta.Dest)
	} else {
		destDir = filepath.Join(projectDir, "site", "src", "shared", "ui", name)
	}

	// Check if the first file exists as a proxy
	if len(meta.Files) > 0 {
		firstFile := filepath.Join(destDir, meta.Files[0])
		if _, err := os.Stat(firstFile); err == nil {
			return true
		}
	}

	return false
}
