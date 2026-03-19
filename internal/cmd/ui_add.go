package cmd

import (
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

func newUIAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <asset>[@version]",
		Short: i18n.T("ui_add_short"),
		Args:  cobra.ExactArgs(1),
		RunE:  runUIAdd,
	}
}

func runUIAdd(cmd *cobra.Command, args []string) error {
	name, version := parseAssetArg(args[0])

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	cfg := config.Default()
	mgr := asset.NewManager(cfg.CacheDir)

	fmt.Println(i18n.T("ui_add_cache"))
	if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
		return fmt.Errorf("update cache: %w", err)
	}

	reg, err := mgr.GetRegistry("ui")
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	// Resolve shortname to full key
	fullName, ok := reg.ResolveAssetName(name)
	if !ok {
		return fmt.Errorf("%s", i18n.Tf("ui_add_not_found", name, name, name))
	}

	installing := i18n.Tf("ui_add_installing", fullName)
	if version != "" {
		installing += "@" + version
	}
	fmt.Println(installing + "...")

	installedResult, err := asset.InstallUIAssetFull(mgr, fullName, version, projectDir)
	if err != nil {
		return fmt.Errorf("install %s: %w", fullName, err)
	}
	installedVer := installedResult.InstalledVersion
	depsInjected := installedResult.DepsInjected

	// Load lock and track installed assets
	lockPath := filepath.Join(projectDir, "gve.lock")
	lf, err := lock.Load(lockPath)
	if err != nil {
		return fmt.Errorf("load gve.lock: %w", err)
	}
	lf.SetUIAsset(fullName, installedVer)

	// Resolve peerDeps recursively
	installed := make(map[string]bool)
	for k := range lf.UI.Assets {
		installed[k] = true
	}
	installed[fullName] = true

	peerDeps, err := asset.ResolvePeerDepsRecursive(mgr, fullName, installed, 5)
	if err != nil {
		fmt.Println(i18n.Tf("ui_add_peer_warn", err))
	}
	for _, dep := range peerDeps {
		fmt.Println(i18n.Tf("ui_add_peer_installing", dep))
		depVer, err := asset.InstallUIAsset(mgr, dep, "", projectDir)
		if err != nil {
			fmt.Println(i18n.Tf("ui_add_peer_fail", dep, err))
			continue
		}
		lf.SetUIAsset(dep, depVer)
		fmt.Println(i18n.Tf("ui_add_peer_ok", dep, depVer))
	}

	if err := lf.Save(lockPath); err != nil {
		return fmt.Errorf("save gve.lock: %w", err)
	}

	// Auto-run npm install if new deps were injected
	if depsInjected {
		siteDir := filepath.Join(projectDir, "site")
		fmt.Println(i18n.T("ui_add_npm_detect"))
		if err := runNodeInstall(siteDir); err != nil {
			fmt.Println(i18n.Tf("ui_add_npm_warn", err))
		}
	}

	fmt.Println(i18n.Tf("ui_add_ok", fullName, installedVer))
	return nil
}

// parseAssetArg splits "button@1.2.0" into ("button", "1.2.0").
// Also handles "ui/button@1.2.0" -> ("ui/button", "1.2.0").
// If no @ is present, version is empty (meaning latest).
func parseAssetArg(arg string) (name, version string) {
	if idx := strings.LastIndex(arg, "@"); idx > 0 {
		return arg[:idx], arg[idx+1:]
	}
	return arg, ""
}

// findProjectRoot walks up from cwd to find a directory containing gve.lock.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "gve.lock")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("%s", i18n.T("common_lock_not_found"))
}
