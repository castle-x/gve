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

func newAPIUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: i18n.T("api_update_short"),
		Long:  i18n.T("api_update_long"),
		RunE:  runAPIUpdate,
	}
}

func runAPIUpdate(cmd *cobra.Command, args []string) error {
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
		fmt.Println(i18n.T("api_update_none"))
		return nil
	}

	mgr := asset.NewManager(cfg.CacheDir)
	fmt.Println(i18n.T("api_update_cache"))
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
			fmt.Println(i18n.Tf("api_update_not_in_reg", name))
			continue
		}

		if entry.Version == info.Latest {
			if asset.APIAssetDirExists(projectDir, name, entry.Version) {
				fmt.Println(i18n.Tf("api_update_already_latest", name, entry.Version))
				skipped++
				continue
			}
			// Files missing even though version matches — reinstall silently
			fmt.Println(i18n.Tf("api_update_missing", name, entry.Version))
			_, err := asset.InstallAPIAsset(mgr, name, entry.Version, projectDir)
			if err != nil {
				fmt.Println(i18n.Tf("api_update_reinstall_fail", name, err))
			} else {
				fmt.Println(i18n.Tf("api_update_reinstall_ok", name, entry.Version))
				upgraded++
			}
			continue
		}

		fmt.Println(i18n.Tf("api_update_available", name, entry.Version, info.Latest))
		fmt.Print(i18n.T("api_update_confirm"))

		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(line)) != "y" {
			fmt.Println(i18n.Tf("api_update_skipped", name))
			skipped++
			continue
		}

		installedVer, err := asset.InstallAPIAsset(mgr, name, info.Latest, projectDir)
		if err != nil {
			fmt.Println(i18n.Tf("api_update_fail", name, err))
			continue
		}

		lf.SetAPIAsset(name, installedVer)
		fmt.Println(i18n.Tf("api_update_ok", name, installedVer))
		upgraded++
	}

	if upgraded > 0 {
		if err := lf.Save(lockPath); err != nil {
			return fmt.Errorf("save gve.lock: %w", err)
		}
	}

	fmt.Println(i18n.Tf("api_update_summary", upgraded, skipped))
	return nil
}
