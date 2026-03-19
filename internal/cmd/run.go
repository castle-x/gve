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

	"github.com/castle-x/gve/internal/i18n"
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
		Short: i18n.T("run_short"),
		Long:  i18n.T("run_long"),
		RunE:  runRun,
	}
	cmd.Flags().BoolP("foreground", "f", false, i18n.T("run_flag_fg"))
	cmd.Flags().IntP("port", "p", 8080, i18n.T("run_flag_port"))
	cmd.Flags().Bool("skip-build", false, i18n.T("run_flag_skip_build"))

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
		return fmt.Errorf("%s", i18n.Tf("run_already_running", pid))
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
			fmt.Println(i18n.T("run_building"))
			if err := doBuild(projectDir); err != nil {
				return err
			}
		} else {
			fmt.Println(i18n.T("run_up_to_date"))
		}
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("%s", i18n.Tf("run_binary_not_found", binaryPath))
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

	fmt.Println(i18n.Tf("run_fg_start", port))
	fmt.Println(i18n.Tf("run_fg_log", logrotate.SymlinkPath(logPath, logPrefix)))
	fmt.Println()

	err = runner.RunCommand(ctx, runner.CommandOpts{
		Name: binaryPath,
		Env:  []string{fmt.Sprintf("PORT=%d", port)},
	}, multiOut, multiOut)

	os.Remove(pidPath)

	if ctx.Err() != nil {
		fmt.Println(i18n.T("run_fg_stopped"))
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

	fmt.Println(i18n.Tf("run_bg_start", port, pid))
	fmt.Println(i18n.Tf("run_bg_log", symlinkPath))
	return nil
}

// --- gve run stop ---

func newRunStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: i18n.T("run_stop_short"),
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
		fmt.Println(i18n.T("run_no_server"))
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidPath)
		return fmt.Errorf("find process %d: %w", pid, err)
	}

	fmt.Println(i18n.Tf("run_stopping", pid))
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
		fmt.Println(i18n.T("run_sigkill"))
		proc.Signal(syscall.SIGKILL)
	}

	os.Remove(pidPath)
	fmt.Println(i18n.T("run_stopped"))
	return nil
}

// --- gve run restart ---

func newRunRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: i18n.T("run_restart_short"),
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
		Short: i18n.T("run_status_short"),
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
		fmt.Println(i18n.T("run_status_stopped"))
		return nil
	}

	logPath := filepath.Join(projectDir, gveDir, logsDir)
	symlink := logrotate.SymlinkPath(logPath, logPrefix)

	fmt.Println(i18n.T("run_status_running"))
	fmt.Println(i18n.Tf("run_status_pid", pid))
	fmt.Println(i18n.Tf("run_status_log", symlink))
	return nil
}

// --- gve run logs ---

func newRunLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: i18n.T("run_logs_short"),
		RunE:  runLogs,
	}
	cmd.Flags().IntP("lines", "n", 50, i18n.T("run_logs_flag_lines"))
	cmd.Flags().BoolP("follow", "f", false, i18n.T("run_logs_flag_follow"))
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
		return fmt.Errorf("%s", i18n.T("run_logs_not_found"))
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
		return fmt.Errorf("%s", i18n.Tf("run_port_in_use", port))
	}
	ln.Close()
	return nil
}

func doBuild(projectDir string) error {
	siteDir := filepath.Join(projectDir, "site")

	ctx := context.Background()

	// Frontend
	fmt.Println(i18n.T("run_build_frontend"))
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
	fmt.Println(i18n.T("run_build_backend"))
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

	fmt.Println(i18n.T("run_build_ok"))
	return nil
}
