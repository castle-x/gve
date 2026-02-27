package logrotate

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	CompressAfterDays = 7
	DeleteAfterDays   = 30
	DateFormat        = "2006-01-02"
)

// Writer is a daily-rotating log writer.
// It creates files like app-2026-02-26.log and maintains
// an app.log symlink pointing to the current file.
type Writer struct {
	dir      string
	prefix   string
	mu       sync.Mutex
	file     *os.File
	currDate string
	nowFunc  func() time.Time // injectable for testing
}

// New creates a rotating log writer.
// Logs are written to dir/{prefix}-{date}.log with a {prefix}.log symlink.
func New(dir, prefix string) (*Writer, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	w := &Writer{
		dir:     dir,
		prefix:  prefix,
		nowFunc: time.Now,
	}
	if err := w.rotate(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := w.nowFunc().Format(DateFormat)
	if today != w.currDate {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	return w.file.Write(p)
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *Writer) rotate() error {
	if w.file != nil {
		w.file.Close()
	}

	today := w.nowFunc().Format(DateFormat)
	logName := fmt.Sprintf("%s-%s.log", w.prefix, today)
	logPath := filepath.Join(w.dir, logName)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	w.file = f
	w.currDate = today

	w.updateSymlink(logName)

	return nil
}

func (w *Writer) updateSymlink(target string) {
	linkPath := filepath.Join(w.dir, w.prefix+".log")
	os.Remove(linkPath)
	os.Symlink(target, linkPath)
}

// Maintain compresses old logs and deletes expired archives.
// Call periodically (e.g., on startup or daily).
func Maintain(dir, prefix string) error {
	return MaintainWithTime(dir, prefix, time.Now())
}

// MaintainWithTime is like Maintain but accepts a custom "now" for testing.
func MaintainWithTime(dir, prefix string, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read log dir: %w", err)
	}

	for _, e := range entries {
		name := e.Name()

		if date, ok := parseLogDate(name, prefix); ok {
			age := now.Sub(date)
			dayAge := int(age.Hours() / 24)

			if dayAge >= DeleteAfterDays {
				os.Remove(filepath.Join(dir, name))
				continue
			}
			if dayAge >= CompressAfterDays && strings.HasSuffix(name, ".log") {
				compressFile(filepath.Join(dir, name))
				continue
			}
		}

		if date, ok := parseGzDate(name, prefix); ok {
			age := now.Sub(date)
			if int(age.Hours()/24) >= DeleteAfterDays {
				os.Remove(filepath.Join(dir, name))
			}
		}
	}

	return nil
}

// parseLogDate extracts the date from "{prefix}-{date}.log".
func parseLogDate(name, prefix string) (time.Time, bool) {
	pfx := prefix + "-"
	sfx := ".log"
	if !strings.HasPrefix(name, pfx) || !strings.HasSuffix(name, sfx) {
		return time.Time{}, false
	}
	dateStr := strings.TrimPrefix(name, pfx)
	dateStr = strings.TrimSuffix(dateStr, sfx)
	t, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// parseGzDate extracts the date from "{prefix}-{date}.log.gz".
func parseGzDate(name, prefix string) (time.Time, bool) {
	pfx := prefix + "-"
	sfx := ".log.gz"
	if !strings.HasPrefix(name, pfx) || !strings.HasSuffix(name, sfx) {
		return time.Time{}, false
	}
	dateStr := strings.TrimPrefix(name, pfx)
	dateStr = strings.TrimSuffix(dateStr, sfx)
	t, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func compressFile(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(path + ".gz")
	if err != nil {
		return err
	}
	defer dst.Close()

	gz := gzip.NewWriter(dst)
	gz.Name = filepath.Base(path)
	if _, err := io.Copy(gz, src); err != nil {
		gz.Close()
		os.Remove(path + ".gz")
		return err
	}
	if err := gz.Close(); err != nil {
		os.Remove(path + ".gz")
		return err
	}

	src.Close()
	os.Remove(path)
	return nil
}

// LogFiles returns sorted log file paths (newest first) for display.
func LogFiles(dir, prefix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	pfx := prefix + "-"
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), pfx) && (strings.HasSuffix(e.Name(), ".log") || strings.HasSuffix(e.Name(), ".log.gz")) {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files, nil
}

// CurrentLogPath returns the path to the current (today's) log file.
func CurrentLogPath(dir, prefix string) string {
	return filepath.Join(dir, fmt.Sprintf("%s-%s.log", prefix, time.Now().Format(DateFormat)))
}

// SymlinkPath returns the path to the stable symlink.
func SymlinkPath(dir, prefix string) string {
	return filepath.Join(dir, prefix+".log")
}
