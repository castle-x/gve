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

func newAPIAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <project>/<resource>[@version]",
		Short: "安装 API 契约",
		Args:  cobra.ExactArgs(1),
		RunE:  runAPIAdd,
	}
}

func runAPIAdd(cmd *cobra.Command, args []string) error {
	resourcePath, version := parseAPIAssetArg(args[0])

	if !strings.Contains(resourcePath, "/") {
		return fmt.Errorf("invalid API resource path %q — expected format: project/resource (e.g. ai-worker/task)", resourcePath)
	}

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	cfg := config.Default()
	mgr := asset.NewManager(cfg.CacheDir)

	fmt.Println("Updating API registry cache...")
	if err := mgr.EnsureCache(cfg.APIRegistry, "api"); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}

	fmt.Printf("Installing %s", resourcePath)
	if version != "" {
		fmt.Printf("@%s", version)
	}
	fmt.Println("...")

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

	fmt.Printf("✓ Installed %s@%s\n", resourcePath, installedVer)
	return nil
}

// parseAPIAssetArg splits "ai-worker/task@v1" into ("ai-worker/task", "v1").
func parseAPIAssetArg(arg string) (resourcePath, version string) {
	if idx := strings.LastIndex(arg, "@"); idx > 0 {
		return arg[:idx], arg[idx+1:]
	}
	return arg, ""
}
