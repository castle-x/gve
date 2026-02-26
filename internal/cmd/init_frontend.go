package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/lock"
)

func initFrontend(projectDir string, cfg *config.Config) error {
	mgr := asset.NewManager(cfg.CacheDir)

	// Clone/pull the UI registry
	if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
		return fmt.Errorf("ensure ui cache: %w", err)
	}

	// Load registry
	reg, err := mgr.GetRegistry("ui")
	if err != nil {
		return fmt.Errorf("load ui registry: %w", err)
	}

	// Get latest base-setup
	ver, assetPath, err := reg.GetLatest("base-setup")
	if err != nil {
		return fmt.Errorf("get base-setup: %w", err)
	}

	// Load meta.json
	metaPath := filepath.Join(mgr.GetAssetDir("ui", assetPath), "meta.json")
	meta, err := asset.LoadMeta(metaPath)
	if err != nil {
		return fmt.Errorf("load base-setup meta: %w", err)
	}

	// Determine destination
	dest := "site"
	if meta.Dest != "" {
		dest = meta.Dest
	}
	destDir := filepath.Join(projectDir, dest)

	// Copy files
	srcDir := mgr.GetAssetDir("ui", assetPath)
	if err := asset.CopyAsset(srcDir, destDir, meta.Files); err != nil {
		return fmt.Errorf("copy base-setup: %w", err)
	}

	// Create placeholder directories
	placeholders := []string{
		filepath.Join(destDir, "src", "views"),
		filepath.Join(destDir, "src", "shared", "ui"),
	}
	for _, dir := range placeholders {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		gitkeep := filepath.Join(dir, ".gitkeep")
		if err := os.WriteFile(gitkeep, nil, 0644); err != nil {
			return fmt.Errorf("create gitkeep: %w", err)
		}
	}

	// Update gve.lock
	lockPath := filepath.Join(projectDir, "gve.lock")
	lf, err := lock.Load(lockPath)
	if err != nil {
		return fmt.Errorf("load gve.lock: %w", err)
	}
	lf.SetUIAsset("base-setup", ver)
	if err := lf.Save(lockPath); err != nil {
		return fmt.Errorf("save gve.lock: %w", err)
	}

	return nil
}
