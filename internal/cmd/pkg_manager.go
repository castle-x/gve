package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/runner"
)

// runNodeInstall runs npm/pnpm install in the given directory.
// It prefers pnpm and falls back to npm if pnpm is unavailable or fails.
func runNodeInstall(dir string) error {
	ctx := context.Background()

	// Try pnpm first
	if _, err := exec.LookPath("pnpm"); err == nil {
		if err := runner.RunCommand(ctx, runner.CommandOpts{
			Name: "pnpm",
			Args: []string{"install"},
			Dir:  dir,
		}, os.Stdout, os.Stderr); err == nil {
			return nil
		}
		fmt.Println(i18n.T("pkg_pnpm_fallback"))
	}

	// Try npm
	if _, err := exec.LookPath("npm"); err == nil {
		if err := runner.RunCommand(ctx, runner.CommandOpts{
			Name: "npm",
			Args: []string{"install"},
			Dir:  dir,
		}, os.Stdout, os.Stderr); err == nil {
			return nil
		}
		return fmt.Errorf("%s", i18n.Tf("pkg_npm_fail", dir))
	}

	return fmt.Errorf("%s", i18n.T("pkg_none_available"))
}

// detectPackageManager detects the preferred package manager for a site directory.
// It checks lock files first, then falls back to exec.LookPath.
func detectPackageManager(siteDir string) string {
	// Check lock files
	if _, err := os.Stat(filepath.Join(siteDir, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(siteDir, "package-lock.json")); err == nil {
		return "npm"
	}

	// Check availability
	if _, err := exec.LookPath("pnpm"); err == nil {
		return "pnpm"
	}
	if _, err := exec.LookPath("npm"); err == nil {
		return "npm"
	}

	return "pnpm" // default, let execution report the error
}
