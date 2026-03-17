package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/lock"
	"github.com/spf13/cobra"
)

func newUISyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [asset]",
		Short: "同步 UI 资产",
		Long:  "将 UI 资产升级到最新版本。指定资产名只同步该资产，否则同步全部。",
		RunE:  runUISync,
	}
}

func runUISync(cmd *cobra.Command, args []string) error {
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
	fmt.Println("Updating UI registry cache...")
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
		entry, ok := lf.UI.Assets[name]
		if !ok {
			fmt.Printf("  %s: not installed, skipping\n", name)
			continue
		}

		info, ok := reg[name]
		if !ok {
			fmt.Printf("  %s: not found in registry\n", name)
			continue
		}

		if entry.Version == info.Latest {
			fmt.Printf("  %s: %s (already latest)\n", name, entry.Version)
			skipped++
			continue
		}

		ve, ok := info.Versions[entry.Version]
		if !ok {
			fmt.Printf("  %s: current version %s not found in registry, skipping\n", name, entry.Version)
			continue
		}

		cacheDir := mgr.GetAssetDir("ui", ve.Path)
		meta, err := asset.LoadMeta(filepath.Join(cacheDir, "meta.json"))
		if err != nil {
			fmt.Printf("  %s: cannot load meta: %v\n", name, err)
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

		fmt.Printf("\n  %s: %s → %s\n", name, entry.Version, info.Latest)

		if hasChanges {
			fmt.Println("  ⚠ Local modifications detected")
			action := promptSyncAction()

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
				fmt.Print("  Proceed with overwrite? [y/n]: ")
				if !confirmYes() {
					fmt.Printf("  Skipped %s\n", name)
					skipped++
					continue
				}
			case "k", "s":
				fmt.Printf("  Skipped %s\n", name)
				skipped++
				continue
			}
		}

		installedVer, err := asset.InstallUIAsset(mgr, name, info.Latest, projectDir)
		if err != nil {
			fmt.Printf("  ✗ Failed to upgrade %s: %v\n", name, err)
			continue
		}

		lf.SetUIAsset(name, installedVer)
		fmt.Printf("  ✓ Upgraded %s to %s\n", name, installedVer)
		upgraded++
	}

	if upgraded > 0 {
		if err := lf.Save(lockPath); err != nil {
			return fmt.Errorf("save gve.lock: %w", err)
		}
	}

	fmt.Printf("\nUI sync complete: %d upgraded, %d skipped\n", upgraded, skipped)
	return nil
}

func promptSyncAction() string {
	fmt.Println("  Options:")
	fmt.Println("    [u] upgrade  — discard local changes, install new version")
	fmt.Println("    [d] diff     — show diff, then decide")
	fmt.Println("    [k] keep     — skip this asset")
	fmt.Println("    [s] skip     — skip this asset")
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
