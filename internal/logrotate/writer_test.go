package logrotate

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriter_CreatesDailyFile(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir, "app")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	w.Write([]byte("hello\n"))

	today := time.Now().Format(DateFormat)
	logFile := filepath.Join(dir, fmt.Sprintf("app-%s.log", today))
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("expected log file %s to exist", logFile)
	}

	data, _ := os.ReadFile(logFile)
	if string(data) != "hello\n" {
		t.Errorf("got %q, want %q", data, "hello\n")
	}
}

func TestWriter_CreatesSymlink(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir, "app")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	w.Write([]byte("test\n"))

	linkPath := filepath.Join(dir, "app.log")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("symlink not found: %v", err)
	}

	today := time.Now().Format(DateFormat)
	expected := fmt.Sprintf("app-%s.log", today)
	if target != expected {
		t.Errorf("symlink target = %q, want %q", target, expected)
	}
}

func TestWriter_RotatesOnDateChange(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir, "app")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	day1 := time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC)

	w.nowFunc = func() time.Time { return day1 }
	w.rotate()
	w.Write([]byte("day1\n"))

	w.nowFunc = func() time.Time { return day2 }
	w.Write([]byte("day2\n"))

	f1 := filepath.Join(dir, "app-2026-02-20.log")
	f2 := filepath.Join(dir, "app-2026-02-21.log")

	d1, _ := os.ReadFile(f1)
	d2, _ := os.ReadFile(f2)

	if !strings.Contains(string(d1), "day1") {
		t.Errorf("day1 log missing expected content, got %q", d1)
	}
	if !strings.Contains(string(d2), "day2") {
		t.Errorf("day2 log missing expected content, got %q", d2)
	}
}

func TestMaintain_CompressOldLogs(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)

	oldDate := now.AddDate(0, 0, -8) // 8 days ago
	oldName := fmt.Sprintf("app-%s.log", oldDate.Format(DateFormat))
	os.WriteFile(filepath.Join(dir, oldName), []byte("old log"), 0644)

	recentDate := now.AddDate(0, 0, -3) // 3 days ago
	recentName := fmt.Sprintf("app-%s.log", recentDate.Format(DateFormat))
	os.WriteFile(filepath.Join(dir, recentName), []byte("recent log"), 0644)

	if err := MaintainWithTime(dir, "app", now); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, oldName)); !os.IsNotExist(err) {
		t.Error("old .log file should have been removed after compression")
	}
	if _, err := os.Stat(filepath.Join(dir, oldName+".gz")); os.IsNotExist(err) {
		t.Error("compressed .gz file should exist")
	}

	if _, err := os.Stat(filepath.Join(dir, recentName)); os.IsNotExist(err) {
		t.Error("recent log should still exist uncompressed")
	}
}

func TestMaintain_DeleteExpiredArchives(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)

	expiredDate := now.AddDate(0, 0, -35) // 35 days ago
	expiredGz := fmt.Sprintf("app-%s.log.gz", expiredDate.Format(DateFormat))

	f, _ := os.Create(filepath.Join(dir, expiredGz))
	gz := gzip.NewWriter(f)
	gz.Write([]byte("expired"))
	gz.Close()
	f.Close()

	if err := MaintainWithTime(dir, "app", now); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, expiredGz)); !os.IsNotExist(err) {
		t.Error("expired .gz archive should have been deleted")
	}
}

func TestMaintain_DeleteExpiredPlainLogs(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)

	expiredDate := now.AddDate(0, 0, -31)
	expiredLog := fmt.Sprintf("app-%s.log", expiredDate.Format(DateFormat))
	os.WriteFile(filepath.Join(dir, expiredLog), []byte("expired"), 0644)

	if err := MaintainWithTime(dir, "app", now); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, expiredLog)); !os.IsNotExist(err) {
		t.Error("expired .log should have been deleted")
	}
}

func TestCompressFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	content := "hello compressed world"
	os.WriteFile(path, []byte(content), 0644)

	if err := compressFile(path); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("original file should be removed after compression")
	}

	gzPath := path + ".gz"
	f, err := os.Open(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()

	data, _ := io.ReadAll(gz)
	if string(data) != content {
		t.Errorf("decompressed = %q, want %q", data, content)
	}
}

func TestLogFiles_SortedNewestFirst(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "app-2026-02-24.log"), nil, 0644)
	os.WriteFile(filepath.Join(dir, "app-2026-02-26.log"), nil, 0644)
	os.WriteFile(filepath.Join(dir, "app-2026-02-25.log"), nil, 0644)
	os.WriteFile(filepath.Join(dir, "unrelated.txt"), nil, 0644)

	files, err := LogFiles(dir, "app")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("got %d files, want 3", len(files))
	}
	if !strings.Contains(files[0], "2026-02-26") {
		t.Errorf("first file should be newest, got %s", files[0])
	}
}

func TestParseLogDate(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		ok     bool
	}{
		{"app-2026-02-26.log", "app", true},
		{"app-2026-02-26.log.gz", "app", false},
		{"other-2026-02-26.log", "app", false},
		{"app.log", "app", false},
	}
	for _, tt := range tests {
		_, ok := parseLogDate(tt.name, tt.prefix)
		if ok != tt.ok {
			t.Errorf("parseLogDate(%q, %q) = _, %v; want %v", tt.name, tt.prefix, ok, tt.ok)
		}
	}
}

func TestParseGzDate(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		ok     bool
	}{
		{"app-2026-02-26.log.gz", "app", true},
		{"app-2026-02-26.log", "app", false},
		{"other-2026-02-26.log.gz", "app", false},
	}
	for _, tt := range tests {
		_, ok := parseGzDate(tt.name, tt.prefix)
		if ok != tt.ok {
			t.Errorf("parseGzDate(%q, %q) = _, %v; want %v", tt.name, tt.prefix, ok, tt.ok)
		}
	}
}
