package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
		fmt.Println("  \u26a0 pnpm install failed, falling back to npm...")
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
		return fmt.Errorf("npm install failed in %s", dir)
	}

	return fmt.Errorf("neither pnpm nor npm is available or working. Please install pnpm (npm install -g pnpm) or npm (https://nodejs.org)")
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
