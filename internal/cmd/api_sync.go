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

func newAPISyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "同步 API 契约",
		Long:  "将 API 契约升级到最新版本。",
		RunE:  runAPISync,
	}
}

func runAPISync(cmd *cobra.Command, args []string) error {
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

	if len(lf.API.Assets) == 0 {
		fmt.Println("No API assets installed.")
		return nil
	}

	mgr := asset.NewManager(cfg.CacheDir)
	fmt.Println("Updating API registry cache...")
	if err := mgr.EnsureCache(cfg.APIRegistry, "api"); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}

	reg, err := mgr.GetRegistry("api")
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	var upgraded, skipped int

	for name, entry := range lf.API.Assets {
		info, ok := reg[name]
		if !ok {
			fmt.Printf("  %s: not found in registry\n", name)
			continue
		}

		if entry.Version == info.Latest {
			if asset.APIAssetDirExists(projectDir, name, entry.Version) {
				fmt.Printf("  %s: %s (already latest)\n", name, entry.Version)
				skipped++
				continue
			}
			// Files missing even though version matches — reinstall silently
			fmt.Printf("  %s: %s (files missing, reinstalling...)\n", name, entry.Version)
			_, err := asset.InstallAPIAsset(mgr, name, entry.Version, projectDir)
			if err != nil {
				fmt.Printf("  ✗ Failed to reinstall %s: %v\n", name, err)
			} else {
				fmt.Printf("  ✓ Reinstalled %s@%s\n", name, entry.Version)
				upgraded++
			}
			continue
		}

		fmt.Printf("\n  %s: %s → %s available\n", name, entry.Version, info.Latest)
		fmt.Print("  Upgrade? [y/n]: ")

		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(line)) != "y" {
			fmt.Printf("  Skipped %s\n", name)
			skipped++
			continue
		}

		installedVer, err := asset.InstallAPIAsset(mgr, name, info.Latest, projectDir)
		if err != nil {
			fmt.Printf("  ✗ Failed to upgrade %s: %v\n", name, err)
			continue
		}

		lf.SetAPIAsset(name, installedVer)
		fmt.Printf("  ✓ Upgraded %s to %s\n", name, installedVer)
		upgraded++
	}

	if upgraded > 0 {
		if err := lf.Save(lockPath); err != nil {
			return fmt.Errorf("save gve.lock: %w", err)
		}
	}

	fmt.Printf("\nAPI sync complete: %d upgraded, %d skipped\n", upgraded, skipped)
	return nil
}
