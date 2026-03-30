package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/lock"
)

func initFrontend(projectDir, projectName string, cfg *config.Config, scaffoldKey string) error {
	mgr := asset.NewManager(cfg.CacheDir)

	// Clone/pull the UI registry
	if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
		return fmt.Errorf("ensure ui cache: %w", err)
	}

	// Load registry
	reg, err := mgr.GetRegistry("ui")
	if err != nil {
		return fmt.Errorf("load ui registry: %w", err)
	}

	// Verify the scaffold exists in registry
	if _, ok := reg[scaffoldKey]; !ok {
		available := reg.ListByCategory("scaffold")
		if len(available) == 0 {
			return fmt.Errorf("%s", i18n.Tf("init_fe_scaffold_not_found", scaffoldKey))
		}
		return fmt.Errorf("%s", i18n.Tf("init_fe_scaffold_list", scaffoldKey, formatScaffoldList(available)))
	}

	// Get latest version
	ver, assetPath, err := reg.GetLatest(scaffoldKey)
	if err != nil {
		return fmt.Errorf("get %s: %w", scaffoldKey, err)
	}

	// Load meta.json
	metaPath := filepath.Join(mgr.GetAssetDir("ui", assetPath), "meta.json")
	meta, err := asset.LoadMeta(metaPath)
	if err != nil {
		return fmt.Errorf("load %s meta: %w", scaffoldKey, err)
	}

	// Determine destination
	dest := "site"
	if meta.Dest != "" {
		dest = meta.Dest
	}
	destDir := filepath.Join(projectDir, dest)

	// Copy files
	srcDir := mgr.GetAssetDir("ui", assetPath)
	if err := asset.CopyAsset(srcDir, destDir, meta.Files); err != nil {
		return fmt.Errorf("copy %s: %w", scaffoldKey, err)
	}

	// Create placeholder directories
	placeholders := []string{
		filepath.Join(destDir, "src", "views"),
		filepath.Join(destDir, "src", "shared", "wk", "ui"),
		filepath.Join(destDir, "src", "shared", "wk", "components"),
		filepath.Join(destDir, "src", "shared", "wk", "hooks"),
		filepath.Join(destDir, "src", "shared", "shadcn"),
	}
	for _, dir := range placeholders {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		gitkeep := filepath.Join(dir, ".gitkeep")
		if err := os.WriteFile(gitkeep, nil, 0644); err != nil {
			return fmt.Errorf("create gitkeep: %w", err)
		}
	}

	// Update gve.lock
	lockPath := filepath.Join(projectDir, "gve.lock")
	lf, err := lock.Load(lockPath)
	if err != nil {
		return fmt.Errorf("load gve.lock: %w", err)
	}
	lf.SetUIAsset(scaffoldKey, ver)
	if err := lf.Save(lockPath); err != nil {
		return fmt.Errorf("save gve.lock: %w", err)
	}

	// Replace brand name placeholders
	if err := replaceBrandName(projectDir, projectName); err != nil {
		return fmt.Errorf("replace brand name: %w", err)
	}

	siteDir := filepath.Join(projectDir, dest)

	// Install npm dependencies
	fmt.Println(i18n.T("init_fe_npm_install"))
	if err := runNodeInstall(siteDir); err != nil {
		return fmt.Errorf("npm install: %w", err)
	}

	// Install shadcn components if specified (skip those already provided by scaffold)
	if len(meta.ShadcnDeps) > 0 {
		shadcnDir := filepath.Join(siteDir, "src", "shared", "shadcn")
		var toInstall []string
		var skipped []string
		for _, comp := range meta.ShadcnDeps {
			compPath := filepath.Join(shadcnDir, comp+".tsx")
			if _, err := os.Stat(compPath); os.IsNotExist(err) {
				toInstall = append(toInstall, comp)
			} else {
				skipped = append(skipped, comp)
			}
		}
		if len(skipped) > 0 {
			fmt.Println(i18n.Tf("init_fe_shadcn_skip", strings.Join(skipped, ", ")))
		}
		if len(toInstall) > 0 {
			fmt.Println(i18n.Tf("init_fe_shadcn", strings.Join(toInstall, ", ")))
			if err := installShadcnComponents(siteDir, toInstall); err != nil {
				fmt.Println(i18n.Tf("init_fe_shadcn_warn", err))
			}
		}
	}

	// Install default UI assets
	if len(meta.DefaultAssets) > 0 {
		fmt.Println(i18n.T("init_fe_assets"))
		var allAssetNames []string // Collect asset names for shadcn resolution
		for _, assetKey := range meta.DefaultAssets {
			// Parse optional @version suffix (e.g. "components/theme-provider@1.0.0")
			assetName, assetVerConstraint := parseAssetArg(assetKey)

			fmt.Println(i18n.Tf("init_fe_asset_arrow", assetKey))

			// Install asset with recursive peerDeps
			installed := make(map[string]bool)
			for k := range lf.UI.Assets {
				installed[k] = true
			}

			assetVer, err := asset.InstallUIAsset(mgr, assetName, assetVerConstraint, projectDir)
			if err != nil {
				fmt.Printf("    ✗ Failed: %v\n", err)
				continue
			}
			lf.SetUIAsset(assetName, assetVer)
			fmt.Printf("    ✓ Installed %s@%s\n", assetName, assetVer)
			allAssetNames = append(allAssetNames, assetName)

			// Resolve and install peerDeps
			installed[assetName] = true
			peerDeps, err := asset.ResolvePeerDepsRecursive(mgr, assetName, installed, 5)
			if err != nil {
				fmt.Printf("    ⚠ peerDeps resolution failed: %v\n", err)
				continue
			}
			for _, dep := range peerDeps {
				depVer, err := asset.InstallUIAsset(mgr, dep, "", projectDir)
				if err != nil {
					fmt.Printf("    ✗ peerDep %s failed: %v\n", dep, err)
					continue
				}
				lf.SetUIAsset(dep, depVer)
				installed[dep] = true
				fmt.Printf("    ✓ peerDep %s@%s\n", dep, depVer)
				allAssetNames = append(allAssetNames, dep)
			}
		}

		// Install shadcn dependencies from all installed components
		if len(allAssetNames) > 0 {
			shadcnNeeded := collectShadcnDeps(mgr, allAssetNames[0], allAssetNames[1:], projectDir)
			if len(shadcnNeeded) > 0 {
				fmt.Println(i18n.Tf("ui_add_shadcn", strings.Join(shadcnNeeded, ", ")))
				if err := installShadcnComponents(siteDir, shadcnNeeded); err != nil {
					fmt.Println(i18n.Tf("ui_add_shadcn_warn", err))
				}
			}
		}

		// Save updated gve.lock
		if err := lf.Save(lockPath); err != nil {
			return fmt.Errorf("save gve.lock: %w", err)
		}

		// Run npm install again for newly injected deps
		fmt.Println("  Refreshing npm dependencies...")
		if err := runNodeInstall(siteDir); err != nil {
			return fmt.Errorf("final npm install: %w", err)
		}
	}

	return nil
}

// formatScaffoldList formats scaffold keys for display.
func formatScaffoldList(scaffolds []string) string {
	result := ""
	for i, s := range scaffolds {
		if i > 0 {
			result += "\n  "
		}
		result += s
	}
	return result
}

// replaceBrandName replaces __PROJECT_NAME__ placeholders in scaffold files
// with the actual project name. Scans site/ root files and all files under site/src/.
func replaceBrandName(projectDir, projectName string) error {
	siteDir := filepath.Join(projectDir, "site")

	// Root-level files
	rootTargets := []string{
		filepath.Join(siteDir, "package.json"),
		filepath.Join(siteDir, "index.html"),
		filepath.Join(siteDir, "app.json"),
	}

	for _, path := range rootTargets {
		if err := replaceInFile(path, "__PROJECT_NAME__", projectName); err != nil {
			return err
		}
	}

	// Walk site/src/ for any remaining placeholders
	srcDir := filepath.Join(siteDir, "src")
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx", ".json", ".html":
			return replaceInFile(path, "__PROJECT_NAME__", projectName)
		}
		return nil
	})
}

// replaceInFile replaces all occurrences of old with new in a file. Skips if file doesn't exist.
func replaceInFile(path, old, new string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}

	original := string(data)
	replaced := strings.ReplaceAll(original, old, new)
	if replaced != original {
		if err := os.WriteFile(path, []byte(replaced), 0644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

// installShadcnComponents installs shadcn components using npx.
// Falls back to pnpm dlx if npx is not available.
func installShadcnComponents(siteDir string, components []string) error {
	args := append([]string{"shadcn@latest", "add"}, components...)
	args = append(args, "--yes", "--overwrite")

	// Try npx first
	if _, err := exec.LookPath("npx"); err == nil {
		cmd := exec.Command("npx", args...)
		cmd.Dir = siteDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// Fallback to pnpm dlx
	if _, err := exec.LookPath("pnpm"); err == nil {
		dlxArgs := append([]string{"dlx"}, args...)
		cmd := exec.Command("pnpm", dlxArgs...)
		cmd.Dir = siteDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pnpm dlx shadcn failed: %w", err)
		}
		return nil
	}

	return fmt.Errorf("neither npx nor pnpm is available for shadcn install")
}
