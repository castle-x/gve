package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/castle-x/gve/internal/asset"
	"github.com/castle-x/gve/internal/config"
	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/lock"
	"github.com/spf13/cobra"
)

func newAPIPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <project>/<resource>",
		Short: i18n.T("api_push_short"),
		Long:  i18n.T("api_push_long"),
		Args:  cobra.ExactArgs(1),
		RunE:  runAPIPush,
	}

	cmd.Flags().String("version", "", i18n.T("api_push_flag_version"))
	cmd.Flags().String("source", "", i18n.T("api_push_flag_source"))
	cmd.Flags().Bool("dry-run", false, i18n.T("api_push_flag_dry_run"))

	return cmd
}

func runAPIPush(cmd *cobra.Command, args []string) error {
	arg := args[0]
	versionFlag, _ := cmd.Flags().GetString("version")
	sourceFlag, _ := cmd.Flags().GetString("source")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Parse project/resource
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("%s", i18n.Tf("api_push_invalid_arg", arg))
	}
	project, resource := parts[0], parts[1]

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	cfg := config.Default()

	// ── Phase 1: Locate source directory ──
	sourceDir, detectedVersion, err := resolveAPISourceDir(project, resource, sourceFlag, versionFlag, projectDir)
	if err != nil {
		return fmt.Errorf("locate source: %w", err)
	}
	fmt.Println(i18n.Tf("api_push_source", sourceDir))

	// ── Phase 2: Determine version ──
	version := versionFlag
	if version == "" {
		version = detectedVersion
	}
	if version == "" {
		return fmt.Errorf("%s", i18n.T("api_push_no_version"))
	}

	// Validate version format
	if !regexp.MustCompile(`^v\d+$`).MatchString(version) {
		return fmt.Errorf("%s", i18n.Tf("api_push_invalid_version", version))
	}
	fmt.Println(i18n.Tf("api_push_version", version))

	// ── Phase 3: Verify .thrift files exist ──
	thriftFiles, err := listThriftFiles(sourceDir)
	if err != nil {
		return fmt.Errorf("scan source: %w", err)
	}
	if len(thriftFiles) == 0 {
		return fmt.Errorf("%s", i18n.Tf("api_push_no_thrift", sourceDir))
	}
	fmt.Println(i18n.Tf("api_push_thrift_files", strings.Join(thriftFiles, ", ")))

	// ── Phase 4: Ensure cache ──
	mgr := asset.NewManager(cfg.CacheDir)
	if !dryRun {
		fmt.Println(i18n.T("api_push_updating_cache"))
		if err := mgr.EnsureCache(cfg.APIRegistry, "api"); err != nil {
			return fmt.Errorf("update cache: %w", err)
		}
	}

	// ── Phase 5: Push to registry ──
	cacheDir := cfg.APICacheDir()
	opts := asset.APIPushOptions{
		CacheDir:  cacheDir,
		Project:   project,
		Resource:  resource,
		Version:   version,
		SourceDir: sourceDir,
		DryRun:    dryRun,
	}

	if err := asset.PushAPIToRegistry(opts); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	// ── Phase 6: Update gve.lock ──
	registryKey := project + "/" + resource
	if !dryRun {
		lockPath := filepath.Join(projectDir, "gve.lock")
		lf, err := lock.Load(lockPath)
		if err != nil {
			fmt.Println(i18n.Tf("api_push_lock_warn", err))
		} else {
			lf.SetAPIAsset(registryKey, version)
			if err := lf.Save(lockPath); err != nil {
				fmt.Println(i18n.Tf("api_push_lock_save_warn", err))
			}
		}
	}

	fmt.Println(i18n.Tf("api_push_ok", registryKey, version))
	return nil
}

// resolveAPISourceDir locates the thrift source directory and optionally detects version.
func resolveAPISourceDir(project, resource, sourceFlag, versionFlag, projectDir string) (dir string, detectedVersion string, err error) {
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
		return abs, "", nil
	}

	// Look in api/{project}/{resource}/ for version directories
	apiDir := filepath.Join(projectDir, "api", project, resource)
	if _, err := os.Stat(apiDir); err != nil {
		return "", "", fmt.Errorf("API directory not found: %s\nUse --source to specify a custom path", apiDir)
	}

	// If version is specified, look for that exact version dir
	if versionFlag != "" {
		vDir := filepath.Join(apiDir, versionFlag)
		if info, err := os.Stat(vDir); err == nil && info.IsDir() {
			return vDir, versionFlag, nil
		}
		return "", "", fmt.Errorf("version directory %s not found in %s", versionFlag, apiDir)
	}

	// Find the highest version directory
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return "", "", fmt.Errorf("read api dir: %w", err)
	}

	vNPattern := regexp.MustCompile(`^v(\d+)$`)
	var versions []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		matches := vNPattern.FindStringSubmatch(e.Name())
		if matches == nil {
			continue
		}
		n, _ := strconv.Atoi(matches[1])
		versions = append(versions, n)
	}

	if len(versions) == 0 {
		return "", "", fmt.Errorf("no version directories (vN) found in %s", apiDir)
	}

	sort.Ints(versions)
	highestVer := fmt.Sprintf("v%d", versions[len(versions)-1])
	return filepath.Join(apiDir, highestVer), highestVer, nil
}

// listThriftFiles returns .thrift file names in a directory.
func listThriftFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".thrift" {
			files = append(files, e.Name())
		}
	}
	return files, nil
}
