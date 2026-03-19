package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/castle-x/gve/internal/i18n"
	"github.com/castle-x/gve/internal/runner"
	"github.com/spf13/cobra"
)

func newDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: i18n.T("dev_short"),
		Long:  i18n.T("dev_long"),
		RunE:  runDev,
	}
	cmd.Flags().IntP("port", "p", 8080, i18n.T("dev_flag_port"))
	cmd.Flags().Int("vite-port", 5173, i18n.T("dev_flag_vite_port"))
	return cmd
}

func runDev(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	siteDir := filepath.Join(projectDir, "site")
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		return fmt.Errorf("%s", i18n.T("dev_site_not_found"))
	}

	// Ensure site/dist/ exists so go:embed all:dist doesn't fail in dev mode.
	distDir := filepath.Join(siteDir, "dist")
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		if err := os.MkdirAll(distDir, 0755); err != nil {
			return fmt.Errorf("create site/dist: %w", err)
		}
		placeholder := filepath.Join(distDir, ".gitkeep")
		if err := os.WriteFile(placeholder, []byte(""), 0644); err != nil {
			return fmt.Errorf("create site/dist/.gitkeep: %w", err)
		}
	}

	// Auto-run npm install if node_modules is missing.
	nodeModules := filepath.Join(siteDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		fmt.Println(i18n.T("dev_node_modules"))
		if err := runNodeInstall(siteDir); err != nil {
			return fmt.Errorf("install dependencies: %w", err)
		}
		fmt.Println()
	}

	port, _ := cmd.Flags().GetInt("port")
	vitePort, _ := cmd.Flags().GetInt("vite-port")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	goWriter := runner.NewPrefixedWriter(os.Stdout, "\033[36m[go]\033[0m")
	goErrWriter := runner.NewPrefixedWriter(os.Stderr, "\033[36m[go]\033[0m")
	viteWriter := runner.NewPrefixedWriter(os.Stdout, "\033[35m[vite]\033[0m")
	viteErrWriter := runner.NewPrefixedWriter(os.Stderr, "\033[35m[vite]\033[0m")

	goOpts := buildGoDevOpts(projectDir, port)
	pm := detectPackageManager(siteDir)
	viteOpts := runner.CommandOpts{
		Name: pm,
		Args: []string{"dev", "--port", fmt.Sprintf("%d", vitePort)},
		Dir:  siteDir,
		Env:  []string{fmt.Sprintf("VITE_BACKEND_TARGET=http://localhost:%d", port)},
	}

	fmt.Println(i18n.T("dev_starting"))
	fmt.Println(i18n.Tf("dev_go_backend", port))
	fmt.Println(i18n.Tf("dev_vite_frontend", vitePort))
	fmt.Println()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(2)

	go func() {
		defer wg.Done()
		defer goWriter.Flush()
		defer goErrWriter.Flush()
		if err := runner.RunCommand(ctx, goOpts, goWriter, goErrWriter); err != nil {
			if ctx.Err() == nil {
				errCh <- fmt.Errorf("go backend exited: %w", err)
				stop()
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer viteWriter.Flush()
		defer viteErrWriter.Flush()
		if err := runner.RunCommand(ctx, viteOpts, viteWriter, viteErrWriter); err != nil {
			if ctx.Err() == nil {
				errCh <- fmt.Errorf("vite dev server exited: %w", err)
				stop()
			}
		}
	}()

	wg.Wait()
	close(errCh)

	for e := range errCh {
		return e
	}

	fmt.Println(i18n.T("dev_stopped"))
	return nil
}

// buildGoDevOpts returns the command opts for the Go backend.
// Uses Air if available, otherwise falls back to go run.
func buildGoDevOpts(projectDir string, port int) runner.CommandOpts {
	env := []string{fmt.Sprintf("PORT=%d", port)}

	if airPath, err := exec.LookPath("air"); err == nil {
		return runner.CommandOpts{
			Name: airPath,
			Dir:  projectDir,
			Env:  env,
		}
	}

	fmt.Println(i18n.T("dev_air_missing"))
	fmt.Println(i18n.T("dev_air_hint"))
	fmt.Println()

	return runner.CommandOpts{
		Name: "go",
		Args: []string{"run", "./cmd/server"},
		Dir:  projectDir,
		Env:  env,
	}
}
