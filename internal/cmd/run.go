package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/castle-x/gve/internal/logrotate"
	"github.com/castle-x/gve/internal/runner"
	"github.com/spf13/cobra"
)

const (
	gveDir         = ".gve"
	logsDir        = "logs"
	pidFileName    = "run.pid"
	logPrefix      = "app"
	shutdownWait   = 5 * time.Second
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "构建并后台运行服务",
		Long:  "智能 build 后以后台进程启动服务。\n使用 --fg 前台运行，Ctrl+C 退出。",
		RunE:  runRun,
	}
	cmd.Flags().BoolP("foreground", "f", false, "前台运行（阻塞终端）")
	cmd.Flags().IntP("port", "p", 8080, "服务端口")
	cmd.Flags().Bool("skip-build", false, "跳过构建，直接启动已有二进制")

	cmd.AddCommand(newRunStopCmd())
	cmd.AddCommand(newRunRestartCmd())
	cmd.AddCommand(newRunStatusCmd())
	cmd.AddCommand(newRunLogsCmd())

	return cmd
}

// --- gve run ---

func runRun(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	port, _ := cmd.Flags().GetInt("port")
	fg, _ := cmd.Flags().GetBool("foreground")
	skipBuild, _ := cmd.Flags().GetBool("skip-build")

	projectName, err := extractProjectName(projectDir)
	if err != nil {
		return fmt.Errorf("detect project name: %w", err)
	}

	binaryPath := filepath.Join(projectDir, "dist", projectName)
	gvePath := filepath.Join(projectDir, gveDir)
	logPath := filepath.Join(gvePath, logsDir)
	pidPath := filepath.Join(gvePath, pidFileName)

	if err := os.MkdirAll(logPath, 0755); err != nil {
		return fmt.Errorf("create .gve/logs: %w", err)
	}

	// Check if already running
	if pid, running := readPIDFile(pidPath); running {
		return fmt.Errorf("server already running (pid: %d) — use 'gve run stop' first", pid)
	}

	// Port conflict check
	if err := checkPort(port); err != nil {
		return err
	}

	// Smart rebuild
	if !skipBuild {
		needsBuild, err := needsRebuild(projectDir, binaryPath)
		if err != nil {
			return fmt.Errorf("check rebuild: %w", err)
		}
		if needsBuild {
			fmt.Println("Building...")
			if err := doBuild(projectDir); err != nil {
				return err
			}
		} else {
			fmt.Println("Binary is up-to-date, skipping build.")
		}
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found at %s — run 'gve build' first", binaryPath)
	}

	// Log maintenance (compress old, delete expired)
	logrotate.Maintain(logPath, logPrefix)

	if fg {
		return runForeground(binaryPath, port, logPath, pidPath)
	}
	return runBackground(binaryPath, port, logPath, pidPath)
}

func runForeground(binaryPath string, port int, logPath, pidPath string) error {
	logWriter, err := logrotate.New(logPath, logPrefix)
	if err != nil {
		return fmt.Errorf("setup log writer: %w", err)
	}
	defer logWriter.Close()

	multiOut := io.MultiWriter(os.Stdout, logWriter)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Running in foreground on :%d (Ctrl+C to stop)\n", port)
	fmt.Printf("Log: %s\n\n", logrotate.SymlinkPath(logPath, logPrefix))

	err = runner.RunCommand(ctx, runner.CommandOpts{
		Name: binaryPath,
		Env:  []string{fmt.Sprintf("PORT=%d", port)},
	}, multiOut, multiOut)

	os.Remove(pidPath)

	if ctx.Err() != nil {
		fmt.Println("\nServer stopped.")
		return nil
	}
	return err
}

func runBackground(binaryPath string, port int, logPath, pidPath string) error {
	logFile := logrotate.CurrentLogPath(logPath, logPrefix)
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	// Update symlink
	symlinkPath := logrotate.SymlinkPath(logPath, logPrefix)
	os.Remove(symlinkPath)
	os.Symlink(filepath.Base(logFile), symlinkPath)

	attr := &os.ProcAttr{
		Dir:   filepath.Dir(binaryPath),
		Env:   append(os.Environ(), fmt.Sprintf("PORT=%d", port)),
		Files: []*os.File{os.Stdin, f, f},
		Sys: &syscall.SysProcAttr{
			Setsid: true,
		},
	}

	proc, err := os.StartProcess(binaryPath, []string{binaryPath}, attr)
	if err != nil {
		f.Close()
		return fmt.Errorf("start process: %w", err)
	}

	pid := proc.Pid
	proc.Release()
	f.Close()

	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}

	fmt.Printf("Server running on :%d (pid: %d)\n", port, pid)
	fmt.Printf("Log: %s\n", symlinkPath)
	return nil
}

// --- gve run stop ---

func newRunStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "停止后台服务",
		RunE:  runStop,
	}
}

func runStop(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}
	return stopServer(projectDir)
}

func stopServer(projectDir string) error {
	pidPath := filepath.Join(projectDir, gveDir, pidFileName)
	pid, running := readPIDFile(pidPath)
	if !running {
		fmt.Println("No running server found.")
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidPath)
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	fmt.Printf("Stopping server (pid: %d)...\n", pid)
	proc.Signal(syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ { // 5 seconds, polling every 100ms
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				close(done)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}()

	<-done

	if err := proc.Signal(syscall.Signal(0)); err == nil {
		fmt.Println("Process did not exit, sending SIGKILL...")
		proc.Signal(syscall.SIGKILL)
	}

	os.Remove(pidPath)
	fmt.Println("Server stopped.")
	return nil
}

// --- gve run restart ---

func newRunRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "重启后台服务",
		RunE:  runRestart,
	}
}

func runRestart(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	stopServer(projectDir)

	return runRun(cmd.Parent(), args)
}

// --- gve run status ---

func newRunStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看服务运行状态",
		RunE:  runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	pidPath := filepath.Join(projectDir, gveDir, pidFileName)
	pid, running := readPIDFile(pidPath)

	if !running {
		fmt.Println("Status: stopped")
		return nil
	}

	logPath := filepath.Join(projectDir, gveDir, logsDir)
	symlink := logrotate.SymlinkPath(logPath, logPrefix)

	fmt.Printf("Status:  running\n")
	fmt.Printf("PID:     %d\n", pid)
	fmt.Printf("Log:     %s\n", symlink)
	return nil
}

// --- gve run logs ---

func newRunLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "查看服务日志",
		RunE:  runLogs,
	}
	cmd.Flags().IntP("lines", "n", 50, "显示最近 N 行")
	cmd.Flags().BoolP("follow", "f", false, "持续跟踪日志输出")
	return cmd
}

func runLogs(cmd *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	logPath := filepath.Join(projectDir, gveDir, logsDir)
	follow, _ := cmd.Flags().GetBool("follow")
	lines, _ := cmd.Flags().GetInt("lines")

	currentLog := logrotate.CurrentLogPath(logPath, logPrefix)
	if _, err := os.Stat(currentLog); os.IsNotExist(err) {
		return fmt.Errorf("no log file found — server may not have been started yet")
	}

	if follow {
		return tailFollow(currentLog)
	}
	return tailLines(currentLog, lines)
}

func tailLines(path string, n int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	start := 0
	if len(allLines) > n {
		start = len(allLines) - n
	}
	for _, line := range allLines[start:] {
		fmt.Println(line)
	}
	return nil
}

func tailFollow(path string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Seek(0, io.SeekEnd)

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			n, err := f.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
			}
			if err == io.EOF {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if err != nil {
				return err
			}
		}
	}
}

// --- helpers ---

func readPIDFile(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	if err := syscall.Kill(pid, 0); err != nil {
		os.Remove(path)
		return 0, false
	}
	return pid, true
}

func checkPort(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("port %d is already in use — stop the existing process or use --port to specify another", port)
	}
	ln.Close()
	return nil
}

func doBuild(projectDir string) error {
	siteDir := filepath.Join(projectDir, "site")

	ctx := context.Background()

	// Frontend
	fmt.Println("  Building frontend...")
	if err := runner.RunCommand(ctx, runner.CommandOpts{
		Name: "pnpm",
		Args: []string{"install", "--frozen-lockfile"},
		Dir:  siteDir,
	}, os.Stdout, os.Stderr); err != nil {
		if err2 := runner.RunCommand(ctx, runner.CommandOpts{
			Name: "pnpm",
			Args: []string{"install"},
			Dir:  siteDir,
		}, os.Stdout, os.Stderr); err2 != nil {
			return fmt.Errorf("pnpm install failed: %w", err2)
		}
	}

	if err := runner.RunCommand(ctx, runner.CommandOpts{
		Name: "pnpm",
		Args: []string{"build"},
		Dir:  siteDir,
	}, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("frontend build failed: %w", err)
	}

	// Backend
	fmt.Println("  Building backend...")
	projectName, _ := extractProjectName(projectDir)
	outputAbs := filepath.Join(projectDir, "dist", projectName)
	os.MkdirAll(filepath.Dir(outputAbs), 0755)

	if err := runner.RunCommand(ctx, runner.CommandOpts{
		Name: "go",
		Args: []string{"build", "-o", outputAbs, "./cmd/server"},
		Dir:  projectDir,
	}, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	fmt.Println("  ✓ Build complete")
	return nil
}
