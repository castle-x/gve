package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// PrefixedWriter wraps an io.Writer and prepends a prefix to each line.
type PrefixedWriter struct {
	out    io.Writer
	prefix string
	mu     sync.Mutex
	buf    bytes.Buffer
}

func NewPrefixedWriter(out io.Writer, prefix string) *PrefixedWriter {
	return &PrefixedWriter{out: out, prefix: prefix}
}

func (w *PrefixedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf.Write(p)

	for {
		line, err := w.buf.ReadBytes('\n')
		if err != nil {
			// Incomplete line — put it back for next write
			w.buf.Write(line)
			break
		}
		fmt.Fprintf(w.out, "%s %s", w.prefix, line)
	}

	return len(p), nil
}

// Flush writes any remaining buffered content (incomplete last line).
func (w *PrefixedWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.buf.Len() > 0 {
		fmt.Fprintf(w.out, "%s %s\n", w.prefix, w.buf.String())
		w.buf.Reset()
	}
}

// CommandOpts configures a command to run.
type CommandOpts struct {
	Name string
	Args []string
	Dir  string
	Env  []string
}

// RunCommand executes a command with context support and prefixed output.
// It returns the exit error (nil on success).
func RunCommand(ctx context.Context, opts CommandOpts, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, opts.Name, opts.Args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

// RunCommandStreaming starts a command and returns immediately.
// The caller should wait on the returned channel for completion.
func RunCommandStreaming(ctx context.Context, opts CommandOpts, stdout, stderr io.Writer) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, opts.Name, opts.Args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
