package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

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

	// Group by category
	categories := []struct {
		label  string
		prefix string
	}{
		{"SCAFFOLD", "scaffold/"},
		{"UI", "ui/"},
		{"COMPONENTS", "components/"},
		{"GLOBAL", "global/"},
	}

	printed := false
	for _, cat := range categories {
		var names []string
		for name := range lf.UI.Assets {
			if strings.HasPrefix(name, cat.prefix) {
				names = append(names, name)
			}
		}
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)

		if printed {
			fmt.Println()
		}
		fmt.Println(cat.label)
		for _, name := range names {
			entry := lf.UI.Assets[name]
			dest := resolveDestPath(name, entry.Version, reg, mgr)
			fmt.Printf("  %-30s v%-10s %s\n", name, entry.Version, dest)
		}
		printed = true
	}

	// Print any uncategorized assets (v1 compat)
	var uncategorized []string
	for name := range lf.UI.Assets {
		hasPrefix := false
		for _, cat := range categories {
			if strings.HasPrefix(name, cat.prefix) {
				hasPrefix = true
				break
			}
		}
		if !hasPrefix {
			uncategorized = append(uncategorized, name)
		}
	}
	if len(uncategorized) > 0 {
		sort.Strings(uncategorized)
		if printed {
			fmt.Println()
		}
		fmt.Println("OTHER")
		for _, name := range uncategorized {
			entry := lf.UI.Assets[name]
			dest := resolveDestPath(name, entry.Version, reg, mgr)
			fmt.Printf("  %-30s v%-10s %s\n", name, entry.Version, dest)
		}
	}

	return nil
}

func resolveDestPath(name, version string, reg asset.Registry, mgr *asset.Manager) string {
	if reg == nil {
		// Fallback: derive from name pattern
		category := asset.InferCategory(name)
		bareName := name
		if idx := strings.Index(name, "/"); idx >= 0 {
			bareName = name[idx+1:]
		}
		path := asset.GetInstallPath(category, bareName, "")
		if path != "" {
			return path + "/"
		}
		return ""
	}

	info, ok := reg[name]
	if !ok {
		// Fallback: derive from name pattern
		category := asset.InferCategory(name)
		bareName := name
		if idx := strings.Index(name, "/"); idx >= 0 {
			bareName = name[idx+1:]
		}
		path := asset.GetInstallPath(category, bareName, "")
		if path != "" {
			return path + "/"
		}
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

	category := meta.Category
	if category == "" {
		category = asset.InferCategory(ve.Path)
	}
	path := asset.GetInstallPath(category, meta.Name, meta.Dest)
	if path != "" {
		return path + "/"
	}
	return ""
}
