package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/lock"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: i18n.T("sync_short"),
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
		fmt.Println(i18n.T("sync_ui"))
		if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
			return fmt.Errorf("update ui cache: %w", err)
		}

		reg, err := mgr.GetRegistry("ui")
		if err != nil {
			return fmt.Errorf("load ui registry: %w", err)
		}

		for name, entry := range lf.UI.Assets {
			// scaffold assets are only used during init, skip during sync
			if strings.HasPrefix(name, "scaffold/") {
				continue
			}

			if assetExists(name, entry.Version, reg, mgr, projectDir) {
				skipped++
				continue
			}

			fmt.Println(i18n.Tf("sync_installing", name, entry.Version))
			_, err := asset.InstallUIAsset(mgr, name, entry.Version, projectDir)
			if err != nil {
				fmt.Println(i18n.Tf("sync_fail", name, err))
				continue
			}
			installed++
		}
	}

	// Sync API assets
	if len(lf.API.Assets) > 0 {
		apiMgr := asset.NewManager(cfg.CacheDir)
		fmt.Println(i18n.T("sync_api"))
		if err := apiMgr.EnsureCache(cfg.APIRegistry, "api"); err != nil {
			return fmt.Errorf("update api cache: %w", err)
		}

		for name, entry := range lf.API.Assets {
			if asset.APIAssetDirExists(projectDir, name, entry.Version) {
				skipped++
				continue
			}

			fmt.Println(i18n.Tf("sync_installing", name, entry.Version))
			_, err := asset.InstallAPIAsset(apiMgr, name, entry.Version, projectDir)
			if err != nil {
				fmt.Println(i18n.Tf("sync_fail", name, err))
				continue
			}
			installed++
		}
	}

	fmt.Println(i18n.Tf("sync_summary", installed, skipped))
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

	// Use v2 category-aware path resolution
	category := meta.Category
	if category == "" {
		category = asset.InferCategory(ve.Path)
	}
	bareName := meta.Name
	if bareName == "" {
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			bareName = name[idx+1:]
		} else {
			bareName = name
		}
	}
	destDir := filepath.Join(projectDir, asset.GetInstallPath(category, bareName, meta.Dest))

	// Check if the first file exists as a proxy
	if len(meta.Files) > 0 {
		firstFile := filepath.Join(destDir, meta.Files[0])
		if _, err := os.Stat(firstFile); err == nil {
			return true
		}
	}

	return false
}
