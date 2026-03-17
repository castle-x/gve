package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
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
			return fmt.Errorf("scaffold %q not found and no scaffolds available in registry", scaffoldKey)
		}
		return fmt.Errorf("scaffold %q not found. Available scaffolds:\n  %s",
			scaffoldKey, formatScaffoldList(available))
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
	fmt.Println("  Installing npm dependencies...")
	if err := runNodeInstall(siteDir); err != nil {
		return fmt.Errorf("npm install: %w", err)
	}

	// Install shadcn components if specified
	if len(meta.ShadcnDeps) > 0 {
		fmt.Printf("  Installing shadcn components: %s\n", strings.Join(meta.ShadcnDeps, ", "))
		if err := installShadcnComponents(siteDir, meta.ShadcnDeps); err != nil {
			fmt.Printf("  ⚠ shadcn install failed: %v (continuing...)\n", err)
		}
	}

	// Install default UI assets
	if len(meta.DefaultAssets) > 0 {
		fmt.Println("  Installing default UI assets...")
		for _, assetKey := range meta.DefaultAssets {
			fmt.Printf("    → %s\n", assetKey)

			// Install asset with recursive peerDeps
			installed := make(map[string]bool)
			for k := range lf.UI.Assets {
				installed[k] = true
			}

			assetVer, err := asset.InstallUIAsset(mgr, assetKey, "", projectDir)
			if err != nil {
				fmt.Printf("    ✗ Failed: %v\n", err)
				continue
			}
			lf.SetUIAsset(assetKey, assetVer)
			fmt.Printf("    ✓ Installed %s@%s\n", assetKey, assetVer)

			// Resolve and install peerDeps
			installed[assetKey] = true
			peerDeps, err := asset.ResolvePeerDepsRecursive(mgr, assetKey, installed, 5)
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
// with the actual project name.
func replaceBrandName(projectDir, projectName string) error {
	targets := []string{
		filepath.Join(projectDir, "site", "package.json"),
		filepath.Join(projectDir, "site", "index.html"),
	}

	for _, path := range targets {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // file may not exist in all scaffolds
			}
			return fmt.Errorf("read %s: %w", path, err)
		}

		original := string(data)
		replaced := strings.ReplaceAll(original, "__PROJECT_NAME__", projectName)
		if replaced != original {
			if err := os.WriteFile(path, []byte(replaced), 0644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}
	}

	return nil
}

// installShadcnComponents installs shadcn components using npx.
// Falls back to pnpm dlx if npx is not available.
func installShadcnComponents(siteDir string, components []string) error {
	args := append([]string{"shadcn@latest", "add"}, components...)
	args = append(args, "--yes")

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
