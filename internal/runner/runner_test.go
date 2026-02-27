package runner

import (
	"bytes"
	"context"
	"runtime"
	"testing"
)

func TestPrefixedWriter_SingleLine(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixedWriter(&buf, "[test]")

	pw.Write([]byte("hello world\n"))

	got := buf.String()
	want := "[test] hello world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixedWriter_MultiLine(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixedWriter(&buf, "[go]")

	pw.Write([]byte("line1\nline2\nline3\n"))

	got := buf.String()
	want := "[go] line1\n[go] line2\n[go] line3\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixedWriter_PartialLines(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixedWriter(&buf, "[vite]")

	pw.Write([]byte("partial"))
	if buf.Len() != 0 {
		t.Error("expected no output for incomplete line")
	}

	pw.Write([]byte(" line\n"))
	got := buf.String()
	want := "[vite] partial line\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixedWriter_Flush(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixedWriter(&buf, "[x]")

	pw.Write([]byte("no newline"))
	if buf.Len() != 0 {
		t.Error("expected no output before flush")
	}

	pw.Flush()
	got := buf.String()
	want := "[x] no newline\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunCommand_Echo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo test not portable on Windows")
	}

	var stdout, stderr bytes.Buffer
	err := RunCommand(context.Background(), CommandOpts{
		Name: "echo",
		Args: []string{"hello"},
	}, &stdout, &stderr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := stdout.String(); got != "hello\n" {
		t.Errorf("stdout = %q, want %q", got, "hello\n")
	}
}

func TestRunCommand_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout, stderr bytes.Buffer
	err := RunCommand(ctx, CommandOpts{
		Name: "sleep",
		Args: []string{"10"},
	}, &stdout, &stderr)

	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestRunCommandStreaming(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo test not portable on Windows")
	}

	var stdout, stderr bytes.Buffer
	cmd, err := RunCommandStreaming(context.Background(), CommandOpts{
		Name: "echo",
		Args: []string{"streaming"},
	}, &stdout, &stderr)

	if err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
	if got := stdout.String(); got != "streaming\n" {
		t.Errorf("stdout = %q, want %q", got, "streaming\n")
	}
}
