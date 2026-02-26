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

func newUIListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出已安装 UI 资产",
		RunE:  runUIList,
	}
}

func runUIList(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	lockPath := filepath.Join(projectDir, "gve.lock")
	lf, err := lock.Load(lockPath)
	if err != nil {
		return fmt.Errorf("load gve.lock: %w", err)
	}

	if len(lf.UI.Assets) == 0 {
		fmt.Println("No UI assets installed.")
		return nil
	}

	// Try to load registry for dest info
	cfg := config.Default()
	mgr := asset.NewManager(cfg.CacheDir)
	reg, _ := mgr.GetRegistry("ui")

	names := make([]string, 0, len(lf.UI.Assets))
	for name := range lf.UI.Assets {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Println("UI Assets:")
	for _, name := range names {
		entry := lf.UI.Assets[name]
		dest := resolveDestPath(name, entry.Version, reg, mgr)
		fmt.Printf("  %-20s v%-10s %s\n", name, entry.Version, dest)
	}

	return nil
}

func resolveDestPath(name, version string, reg asset.Registry, mgr *asset.Manager) string {
	if reg == nil {
		return ""
	}

	info, ok := reg[name]
	if !ok {
		return ""
	}

	ve, ok := info.Versions[version]
	if !ok {
		return ""
	}

	metaPath := filepath.Join(mgr.GetAssetDir("ui", ve.Path), "meta.json")
	meta, err := asset.LoadMeta(metaPath)
	if err != nil {
		return ""
	}

	if meta.Dest != "" {
		return meta.Dest + "/"
	}
	return "site/src/shared/ui/" + name + "/"
}
