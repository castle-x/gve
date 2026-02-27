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

	"github.com/castle-x/gve/internal/runner"
	"github.com/spf13/cobra"
)

func newDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "启动开发服务器（Go + Vite）",
		Long:  "并发启动 Go 后端和 Vite 前端开发服务器。\nGo 后端优先使用 Air 热重载，未安装则 fallback 到 go run。",
		RunE:  runDev,
	}
	cmd.Flags().IntP("port", "p", 8080, "Go 后端端口")
	cmd.Flags().Int("vite-port", 5173, "Vite 开发服务器端口")
	return cmd
}

func runDev(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	siteDir := filepath.Join(projectDir, "site")
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		return fmt.Errorf("site/ directory not found — run 'gve init' first")
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
	viteOpts := runner.CommandOpts{
		Name: "pnpm",
		Args: []string{"dev", "--port", fmt.Sprintf("%d", vitePort)},
		Dir:  siteDir,
	}

	fmt.Printf("Starting dev server...\n")
	fmt.Printf("  Go backend:  http://localhost:%d\n", port)
	fmt.Printf("  Vite frontend: http://localhost:%d\n", vitePort)
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

	fmt.Println("\nDev server stopped.")
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

	fmt.Println("  ℹ Air not found, using 'go run' (no hot reload)")
	fmt.Println("    Install Air: go install github.com/air-verse/air@latest")
	fmt.Println()

	return runner.CommandOpts{
		Name: "go",
		Args: []string{"run", "./cmd/server"},
		Dir:  projectDir,
		Env:  env,
	}
}
