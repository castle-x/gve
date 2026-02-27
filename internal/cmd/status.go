package cmd

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/lock"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "显示资产状态",
		Long:  "对比 gve.lock 与资产库 registry，显示每个已安装资产的版本及可用更新。",
		RunE:  runAssetStatus,
	}
}

func runAssetStatus(cmd *cobra.Command, args []string) error {
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

	if len(lf.UI.Assets) == 0 && len(lf.API.Assets) == 0 {
		fmt.Println("No assets installed.")
		return nil
	}

	mgr := asset.NewManager(cfg.CacheDir)

	if len(lf.UI.Assets) > 0 {
		fmt.Println("[UI]")
		if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
			fmt.Printf("  ⚠ Cannot update UI registry: %v\n", err)
		} else {
			uiReg, err := mgr.GetRegistry("ui")
			if err != nil {
				fmt.Printf("  ⚠ Cannot load UI registry: %v\n", err)
			} else {
				printAssetStatus(lf.UI.Assets, uiReg)
			}
		}
		fmt.Println()
	}

	if len(lf.API.Assets) > 0 {
		fmt.Println("[API]")
		if err := mgr.EnsureCache(cfg.APIRegistry, "api"); err != nil {
			fmt.Printf("  ⚠ Cannot update API registry: %v\n", err)
		} else {
			apiReg, err := mgr.GetRegistry("api")
			if err != nil {
				fmt.Printf("  ⚠ Cannot load API registry: %v\n", err)
			} else {
				printAssetStatus(lf.API.Assets, apiReg)
			}
		}
	}

	return nil
}

func printAssetStatus(assets map[string]lock.AssetEntry, reg asset.Registry) {
	names := make([]string, 0, len(assets))
	for name := range assets {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		entry := assets[name]
		info, ok := reg[name]
		if !ok {
			fmt.Printf("  %-30s %s (not in registry)\n", name, entry.Version)
			continue
		}
		if entry.Version == info.Latest {
			fmt.Printf("  %-30s %s (latest)\n", name, entry.Version)
		} else {
			fmt.Printf("  %-30s %s → %s available\n", name, entry.Version, info.Latest)
		}
	}
}
