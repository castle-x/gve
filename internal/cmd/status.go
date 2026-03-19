package cmd

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/lock"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: i18n.T("status_short"),
		Long:  i18n.T("status_long"),
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
		fmt.Println(i18n.T("status_none"))
		return nil
	}

	mgr := asset.NewManager(cfg.CacheDir)

	if len(lf.UI.Assets) > 0 {
		fmt.Println(i18n.T("status_ui_header"))
		if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
			fmt.Println(i18n.Tf("status_ui_cache_warn", err))
		} else {
			uiReg, err := mgr.GetRegistry("ui")
			if err != nil {
				fmt.Println(i18n.Tf("status_ui_reg_warn", err))
			} else {
				printAssetStatus(lf.UI.Assets, uiReg)
			}
		}
		fmt.Println()
	}

	if len(lf.API.Assets) > 0 {
		fmt.Println(i18n.T("status_api_header"))
		if err := mgr.EnsureCache(cfg.APIRegistry, "api"); err != nil {
			fmt.Println(i18n.Tf("status_api_cache_warn", err))
		} else {
			apiReg, err := mgr.GetRegistry("api")
			if err != nil {
				fmt.Println(i18n.Tf("status_api_reg_warn", err))
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
			fmt.Printf("  %-30s %s %s\n", name, entry.Version, i18n.T("status_not_in_reg"))
			continue
		}
		if entry.Version == info.Latest {
			fmt.Printf("  %-30s %s %s\n", name, entry.Version, i18n.T("status_latest"))
		} else {
			fmt.Printf("  %-30s %s %s\n", name, entry.Version, i18n.Tf("status_available", info.Latest))
		}
	}
}
