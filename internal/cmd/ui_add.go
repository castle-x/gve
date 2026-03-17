package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/lock"
	"github.com/spf13/cobra"
)

func newUIAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <asset>[@version]",
		Short: "安装 UI 资产",
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

	fmt.Printf("Updating UI registry cache...\n")
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
		return fmt.Errorf("asset %q not found in registry. Try: gve ui add ui/%s or components/%s", name, name, name)
	}

	fmt.Printf("Installing %s", fullName)
	if version != "" {
		fmt.Printf("@%s", version)
	}
	fmt.Println("...")

	installedVer, err := asset.InstallUIAsset(mgr, fullName, version, projectDir)
	if err != nil {
		return fmt.Errorf("install %s: %w", fullName, err)
	}

	// Load lock and track installed assets
	lockPath := filepath.Join(projectDir, "gve.lock")
	lf, err := lock.Load(lockPath)
	if err != nil {
		return fmt.Errorf("load gve.lock: %w", err)
	}
	lf.SetUIAsset(fullName, installedVer)

	// Resolve peerDeps
	assetPath := ""
	if version == "" || version == "latest" {
		_, assetPath, _ = reg.GetLatest(fullName)
	} else {
		assetPath, _ = reg.GetVersion(fullName, version)
	}
	if assetPath != "" {
		meta, _ := asset.LoadMeta(filepath.Join(mgr.GetAssetDir("ui", assetPath), "meta.json"))
		if meta != nil && len(meta.PeerDeps) > 0 {
			installed := make(map[string]bool)
			for k := range lf.UI.Assets {
				installed[k] = true
			}
			installed[fullName] = true

			missing := asset.ResolvePeerDeps(meta, installed)
			for _, dep := range missing {
				fmt.Printf("  → peerDep %s: installing...\n", dep)
				depVer, err := asset.InstallUIAsset(mgr, dep, "", projectDir)
				if err != nil {
					fmt.Printf("  → peerDep %s: failed: %v\n", dep, err)
					continue
				}
				lf.SetUIAsset(dep, depVer)
				fmt.Printf("  → peerDep %s: installed v%s\n", dep, depVer)
			}
		}
	}

	if err := lf.Save(lockPath); err != nil {
		return fmt.Errorf("save gve.lock: %w", err)
	}

	fmt.Printf("✓ Installed %s@%s\n", fullName, installedVer)
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

	return "", fmt.Errorf("gve.lock not found — are you inside a GVE project?")
}
