package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/lock"
	"github.com/castle-x/gve/internal/semver"
	"github.com/spf13/cobra"
)

func newUIPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <name>",
		Short: i18n.T("ui_push_short"),
		Long:  i18n.T("ui_push_long"),
		Args:  cobra.ExactArgs(1),
		RunE:  runUIPush,
	}

	cmd.Flags().String("version", "", i18n.T("ui_push_flag_version"))
	cmd.Flags().String("source", "", i18n.T("ui_push_flag_source"))
	cmd.Flags().String("desc", "", i18n.T("ui_push_flag_desc"))
	cmd.Flags().Bool("dry-run", false, i18n.T("ui_push_flag_dry_run"))

	return cmd
}

func runUIPush(cmd *cobra.Command, args []string) error {
	name := args[0]
	versionFlag, _ := cmd.Flags().GetString("version")
	sourceFlag, _ := cmd.Flags().GetString("source")
	descFlag, _ := cmd.Flags().GetString("desc")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	cfg := config.Default()

	// ── Phase 1: Locate source directory ──
	sourceDir, category, err := resolveSourceDir(name, sourceFlag, projectDir)
	if err != nil {
		return fmt.Errorf("locate source: %w", err)
	}
	fmt.Println(i18n.Tf("ui_push_source", sourceDir))

	// ── Phase 2: Scan TSX imports ──
	fmt.Println(i18n.T("ui_push_scanning"))
	scanResult, err := asset.ScanDir(sourceDir)
	if err != nil {
		return fmt.Errorf("scan imports: %w", err)
	}
	fmt.Println(i18n.Tf("ui_push_scanned", len(scanResult.ScannedFiles)))
	if len(scanResult.Deps) > 0 {
		fmt.Println(i18n.Tf("ui_push_npm_deps", strings.Join(scanResult.Deps, ", ")))
	}
	if len(scanResult.PeerDeps) > 0 {
		fmt.Println(i18n.Tf("ui_push_peer_deps", strings.Join(scanResult.PeerDeps, ", ")))
	}

	// ── Phase 3: Determine version ──
	version, err := resolveVersion(name, category, versionFlag, projectDir, cfg)
	if err != nil {
		return fmt.Errorf("resolve version: %w", err)
	}
	fmt.Println(i18n.Tf("ui_push_version", version))

	// ── Phase 4: Build meta.json ──
	files, err := collectFiles(sourceDir)
	if err != nil {
		return fmt.Errorf("collect files: %w", err)
	}

	meta := &asset.Meta{
		Name:        name,
		Version:     version,
		Category:    inferMetaCategory(category),
		Description: descFlag,
		Files:       files,
		Deps:        scanResult.Deps,
		PeerDeps:    scanResult.PeerDeps,
	}

	// Try to preserve description from existing meta
	if descFlag == "" {
		existingMeta := tryLoadExistingMeta(sourceDir)
		if existingMeta != nil && existingMeta.Description != "" {
			meta.Description = existingMeta.Description
		}
	}

	fmt.Println(i18n.Tf("ui_push_meta", len(meta.Files), len(meta.Deps), len(meta.PeerDeps)))

	// ── Phase 5: Publish to registry cache ──
	cacheDir := cfg.UICacheDir()

	// Ensure cache exists
	mgr := asset.NewManager(cfg.CacheDir)
	fmt.Println(i18n.T("ui_push_updating_cache"))
	if !dryRun {
		if err := mgr.EnsureCache(cfg.UIRegistry, "ui"); err != nil {
			return fmt.Errorf("update cache: %w", err)
		}
	}

	// Check peerDeps exist in registry
	if len(scanResult.PeerDeps) > 0 {
		regPath := filepath.Join(cacheDir, "registry.json")
		if reg, err := asset.LoadRegistry(regPath); err == nil {
			for _, dep := range scanResult.PeerDeps {
				if _, ok := reg[dep]; !ok {
					fmt.Println(i18n.Tf("ui_push_peer_warn", dep))
				}
			}
		}
	}

	opts := asset.PushOptions{
		CacheDir:  cacheDir,
		Category:  category,
		Name:      name,
		Version:   version,
		SourceDir: sourceDir,
		Meta:      meta,
		DryRun:    dryRun,
	}

	if err := asset.PushToRegistry(opts); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	// ── Phase 6: Update gve.lock ──
	registryKey := category + "/" + name
	if !dryRun {
		lockPath := filepath.Join(projectDir, "gve.lock")
		lf, err := lock.Load(lockPath)
		if err != nil {
			fmt.Println(i18n.Tf("ui_push_lock_warn", err))
		} else {
			lf.SetUIAsset(registryKey, version)
			if err := lf.Save(lockPath); err != nil {
				fmt.Println(i18n.Tf("ui_push_lock_save_warn", err))
			}
		}
	}

	fmt.Println(i18n.Tf("ui_push_ok", registryKey, version))
	return nil
}

// resolveSourceDir locates the component source directory.
func resolveSourceDir(name, sourceFlag, projectDir string) (dir string, category string, err error) {
	if sourceFlag != "" {
		abs, err := filepath.Abs(sourceFlag)
		if err != nil {
			return "", "", err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", "", fmt.Errorf("source dir %q: %w", abs, err)
		}
		if !info.IsDir() {
			return "", "", fmt.Errorf("source %q is not a directory", abs)
		}
		// Try to infer category from path
		category = inferCategoryFromPath(abs)
		if category == "" {
			category = "ui" // default
		}
		return abs, category, nil
	}

	// Try standard project locations
	candidates := []struct {
		path     string
		category string
	}{
		{filepath.Join(projectDir, "site", "src", "shared", "wk", "ui", name), "ui"},
		{filepath.Join(projectDir, "site", "src", "shared", "wk", "components", name), "components"},
		{filepath.Join(projectDir, "site", "src", "shared", "wk", "hooks", name), "hooks"},
	}

	for _, c := range candidates {
		info, err := os.Stat(c.path)
		if err == nil && info.IsDir() {
			return c.path, c.category, nil
		}
	}

	return "", "", fmt.Errorf("%s", i18n.Tf("ui_push_not_found", name, name, name, name))
}

// resolveVersion determines the version to publish.
func resolveVersion(name, category, versionFlag, projectDir string, cfg *config.Config) (string, error) {
	if versionFlag != "" {
		// Validate it's a valid semver
		if _, err := semver.Parse(versionFlag); err != nil {
			return "", fmt.Errorf("invalid version %q: %w", versionFlag, err)
		}
		return versionFlag, nil
	}

	registryKey := category + "/" + name

	// Try gve.lock
	lockPath := filepath.Join(projectDir, "gve.lock")
	if lf, err := lock.Load(lockPath); err == nil {
		if v, ok := lf.GetUIAsset(registryKey); ok {
			bumped, err := semver.BumpPatch(v)
			if err == nil {
				return bumped, nil
			}
		}
	}

	// Try existing meta.json in cache
	cacheDir := cfg.UICacheDir()
	regPath := filepath.Join(cacheDir, "registry.json")
	if reg, err := asset.LoadRegistry(regPath); err == nil {
		if info, ok := reg[registryKey]; ok && info.Latest != "" {
			bumped, err := semver.BumpPatch(info.Latest)
			if err == nil {
				return bumped, nil
			}
		}
	}

	// Fallback
	return "1.0.0", nil
}

// collectFiles lists all files in dir relative to dir (non-recursive excludes meta.json).
func collectFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Skip meta.json — it's generated, not a source file
		if rel == "meta.json" {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}

// inferCategoryFromPath tries to determine category from the file path.
func inferCategoryFromPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, p := range parts {
		if p == "wk" && i+1 < len(parts) {
			switch parts[i+1] {
			case "ui":
				return "ui"
			case "components":
				return "components"
			case "hooks":
				return "hooks"
			}
		}
	}
	return ""
}

// inferMetaCategory converts directory category to meta.json category value.
func inferMetaCategory(category string) string {
	switch category {
	case "components":
		return "component"
	case "hooks":
		return "hook"
	default:
		return category
	}
}

// tryLoadExistingMeta attempts to load meta.json from a directory, returning nil on failure.
func tryLoadExistingMeta(dir string) *asset.Meta {
	m, _ := asset.LoadMeta(filepath.Join(dir, "meta.json"))
	return m
}

// saveMeta writes a Meta as JSON to the given path (used for dry-run preview).
func saveMeta(m *asset.Meta, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
