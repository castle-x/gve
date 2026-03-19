package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/lock"
	"github.com/spf13/cobra"
)

func newAPIAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <project>/<resource>[@version]",
		Short: i18n.T("api_add_short"),
		Args:  cobra.ExactArgs(1),
		RunE:  runAPIAdd,
	}
}

func runAPIAdd(cmd *cobra.Command, args []string) error {
	resourcePath, version := parseAPIAssetArg(args[0])

	if !strings.Contains(resourcePath, "/") {
		return fmt.Errorf("%s", i18n.Tf("api_add_invalid", resourcePath))
	}

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	cfg := config.Default()
	mgr := asset.NewManager(cfg.CacheDir)

	fmt.Println(i18n.T("api_add_cache"))
	if err := mgr.EnsureCache(cfg.APIRegistry, "api"); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}

	installing := i18n.Tf("api_add_installing", resourcePath)
	if version != "" {
		installing += "@" + version
	}
	fmt.Println(installing + "...")

	installedVer, err := asset.InstallAPIAsset(mgr, resourcePath, version, projectDir)
	if err != nil {
		return fmt.Errorf("install %s: %w", resourcePath, err)
	}

	lockPath := filepath.Join(projectDir, "gve.lock")
	lf, err := lock.Load(lockPath)
	if err != nil {
		return fmt.Errorf("load gve.lock: %w", err)
	}
	lf.SetAPIAsset(resourcePath, installedVer)
	if err := lf.Save(lockPath); err != nil {
		return fmt.Errorf("save gve.lock: %w", err)
	}

	fmt.Println(i18n.Tf("api_add_ok", resourcePath, installedVer))
	return nil
}

// parseAPIAssetArg splits "ai-worker/task@v1" into ("ai-worker/task", "v1").
func parseAPIAssetArg(arg string) (resourcePath, version string) {
	if idx := strings.LastIndex(arg, "@"); idx > 0 {
		return arg[:idx], arg[idx+1:]
	}
	return arg, ""
}
