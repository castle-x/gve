package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/runner"
	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: i18n.T("build_short"),
		Long:  i18n.T("build_long"),
		RunE:  runBuild,
	}
	cmd.Flags().StringP("output", "o", "", i18n.T("build_flag_output"))
	cmd.Flags().String("os", "", i18n.T("build_flag_os"))
	cmd.Flags().String("arch", "", i18n.T("build_flag_arch"))
	return cmd
}

func runBuild(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	siteDir := filepath.Join(projectDir, "site")
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		return fmt.Errorf("%s", i18n.T("build_site_not_found"))
	}

	projectName, err := extractProjectName(projectDir)
	if err != nil {
		return fmt.Errorf("detect project name: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "" {
		output = filepath.Join("dist", projectName)
	}

	targetOS, _ := cmd.Flags().GetString("os")
	targetArch, _ := cmd.Flags().GetString("arch")

	if targetOS == "windows" && !strings.HasSuffix(output, ".exe") {
		output += ".exe"
	}

	ctx := context.Background()

	// Step 1: Frontend build
	fmt.Println(i18n.T("build_frontend"))
	if err := runNodeInstall(siteDir); err != nil {
		return fmt.Errorf("install dependencies: %w", err)
	}

	pm := detectPackageManager(siteDir)
	if err := runner.RunCommand(ctx, runner.CommandOpts{
		Name: pm,
		Args: []string{"build"},
		Dir:  siteDir,
	}, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("frontend build failed: %w", err)
	}

	distDir := filepath.Join(siteDir, "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		return fmt.Errorf("%s", i18n.T("build_frontend_no_dist"))
	}
	fmt.Println(i18n.T("build_frontend_ok"))

	// Step 2: Go backend build
	fmt.Println(i18n.T("build_backend"))

	outputAbs := output
	if !filepath.IsAbs(output) {
		outputAbs = filepath.Join(projectDir, output)
	}
	if err := os.MkdirAll(filepath.Dir(outputAbs), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	buildEnv := buildCrossCompileEnv(targetOS, targetArch)
	if err := runner.RunCommand(ctx, runner.CommandOpts{
		Name: "go",
		Args: []string{"build", "-o", outputAbs, "./cmd/server"},
		Dir:  projectDir,
		Env:  buildEnv,
	}, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	info, err := os.Stat(outputAbs)
	if err != nil {
		return fmt.Errorf("stat output: %w", err)
	}

	fmt.Println("  ✓ Backend build complete")
	fmt.Println()
	fmt.Printf("Build successful!\n")
	fmt.Printf("  Binary: %s\n", output)
	fmt.Printf("  Size:   %s\n", formatSize(info.Size()))

	return nil
}

// extractProjectName reads the project name from go.mod module path
// or falls back to the directory name.
func extractProjectName(projectDir string) (string, error) {
	goModPath := filepath.Join(projectDir, "go.mod")
	f, err := os.Open(goModPath)
	if err != nil {
		return filepath.Base(projectDir), nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			mod := strings.TrimPrefix(line, "module ")
			mod = strings.TrimSpace(mod)
			parts := strings.Split(mod, "/")
			return parts[len(parts)-1], nil
		}
	}

	return filepath.Base(projectDir), nil
}

func buildCrossCompileEnv(targetOS, targetArch string) []string {
	var env []string
	if targetOS != "" {
		env = append(env, "GOOS="+targetOS)
	}
	if targetArch != "" {
		env = append(env, "GOARCH="+targetArch)
	}
	return env
}

func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

