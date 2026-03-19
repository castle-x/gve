package cmd

import (
	"bufio"
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

func newUIUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [asset]",
		Short: i18n.T("ui_update_short"),
		Long:  i18n.T("ui_update_long"),
		RunE:  runUIUpdate,
	}
}

func runUIUpdate(cmd *cobra.Command, args []string) error {
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

	mgr := asset.NewManager(cfg.CacheDir)
	fmt.Println(i18n.T("ui_update_cache"))
	if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}

	reg, err := mgr.GetRegistry("ui")
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	var targets []string
	if len(args) > 0 {
		// Resolve shortname (e.g. "spinner" → "ui/spinner")
		name := args[0]
		if resolved, ok := reg.ResolveAssetName(name); ok {
			name = resolved
		}
		targets = []string{name}
	} else {
		for name := range lf.UI.Assets {
			targets = append(targets, name)
		}
	}

	var upgraded, skipped int

	for _, name := range targets {
		// scaffold assets are only used during init, skip during update
		if strings.HasPrefix(name, "scaffold/") {
			continue
		}

		entry, ok := lf.UI.Assets[name]
		if !ok {
			fmt.Println(i18n.Tf("ui_update_not_installed", name))
			continue
		}

		info, ok := reg[name]
		if !ok {
			fmt.Println(i18n.Tf("ui_update_not_in_reg", name))
			continue
		}

		if entry.Version == info.Latest {
			fmt.Println(i18n.Tf("ui_update_already_latest", name, entry.Version))
			skipped++
			continue
		}

		ve, ok := info.Versions[entry.Version]
		if !ok {
			fmt.Println(i18n.Tf("ui_update_ver_not_found", name, entry.Version))
			continue
		}

		cacheDir := mgr.GetAssetDir("ui", ve.Path)
		meta, err := asset.LoadMeta(filepath.Join(cacheDir, "meta.json"))
		if err != nil {
			fmt.Println(i18n.Tf("ui_update_meta_fail", name, err))
			continue
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
		localDir := filepath.Join(projectDir, asset.GetInstallPath(category, bareName, meta.Dest))

		hasChanges := asset.HasLocalChanges(localDir, cacheDir, meta.Files)

		fmt.Println(i18n.Tf("ui_update_arrow", name, entry.Version, info.Latest))

		if hasChanges {
			fmt.Println(i18n.T("ui_update_local_mod"))
			action := promptUpdateAction()

			switch action {
			case "o", "u":
				// overwrite / upgrade
			case "d":
				diffs, _ := asset.DiffAsset(localDir, cacheDir, meta.Files)
				for _, d := range diffs {
					if d.Status != "unchanged" {
						fmt.Println(d.Diff)
					}
				}
				fmt.Print(i18n.T("ui_update_confirm"))
				if !confirmYes() {
					fmt.Println(i18n.Tf("ui_update_skipped", name))
					skipped++
					continue
				}
			case "k", "s":
				fmt.Println(i18n.Tf("ui_update_skipped", name))
				skipped++
				continue
			}
		}

		installedVer, err := asset.InstallUIAsset(mgr, name, info.Latest, projectDir)
		if err != nil {
			fmt.Println(i18n.Tf("ui_update_fail", name, err))
			continue
		}

		lf.SetUIAsset(name, installedVer)
		fmt.Println(i18n.Tf("ui_update_ok", name, installedVer))
		upgraded++
	}

	if upgraded > 0 {
		if err := lf.Save(lockPath); err != nil {
			return fmt.Errorf("save gve.lock: %w", err)
		}
	}

	fmt.Println(i18n.Tf("ui_update_summary", upgraded, skipped))
	return nil
}

func promptUpdateAction() string {
	fmt.Println(i18n.T("ui_update_options"))
	fmt.Println(i18n.T("ui_update_opt_upgrade"))
	fmt.Println(i18n.T("ui_update_opt_diff"))
	fmt.Println(i18n.T("ui_update_opt_keep"))
	fmt.Println(i18n.T("ui_update_opt_skip"))
	fmt.Print("  > ")

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line))
}

func confirmYes() bool {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line)) == "y"
}
