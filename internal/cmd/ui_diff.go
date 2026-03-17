package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/lock"
	"github.com/spf13/cobra"
)

func newUIDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <asset>",
		Short: "查看资产差异",
		Long:  "对比本地资产文件与资产库对应版本的差异。",
		Args:  cobra.ExactArgs(1),
		RunE:  runUIDiff,
	}
}

func runUIDiff(cmd *cobra.Command, args []string) error {
	assetName := args[0]

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

	version, ok := lf.GetUIAsset(assetName)
	if !ok {
		return fmt.Errorf("asset %q not found in gve.lock — is it installed?", assetName)
	}

	mgr := asset.NewManager(cfg.CacheDir)
	if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}

	reg, err := mgr.GetRegistry("ui")
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	info, ok := reg[assetName]
	if !ok {
		return fmt.Errorf("asset %q not found in registry", assetName)
	}

	ve, ok := info.Versions[version]
	if !ok {
		return fmt.Errorf("version %s of %q not found in registry", version, assetName)
	}

	cacheDir := mgr.GetAssetDir("ui", ve.Path)
	meta, err := asset.LoadMeta(filepath.Join(cacheDir, "meta.json"))
	if err != nil {
		return fmt.Errorf("load meta: %w", err)
	}

	// Use v2 category-aware path resolution
	category := meta.Category
	if category == "" {
		category = asset.InferCategory(ve.Path)
	}
	bareName := meta.Name
	if bareName == "" {
		// Extract from asset name
		if idx := strings.LastIndex(assetName, "/"); idx >= 0 {
			bareName = assetName[idx+1:]
		} else {
			bareName = assetName
		}
	}
	installPath := asset.GetInstallPath(category, bareName, meta.Dest)
	localDir := filepath.Join(projectDir, installPath)

	diffs, err := asset.DiffAsset(localDir, cacheDir, meta.Files)
	if err != nil {
		return fmt.Errorf("diff: %w", err)
	}

	hasChanges := false
	for _, d := range diffs {
		switch d.Status {
		case "unchanged":
			continue
		case "modified":
			fmt.Printf("\033[33mM %s\033[0m\n", d.File)
			fmt.Println(d.Diff)
			hasChanges = true
		case "deleted":
			fmt.Printf("\033[31mD %s\033[0m\n", d.File)
			hasChanges = true
		case "added":
			fmt.Printf("\033[32mA %s\033[0m\n", d.File)
			hasChanges = true
		}
	}

	if !hasChanges {
		fmt.Printf("%s@%s: no local changes\n", assetName, version)
	}

	return nil
}
