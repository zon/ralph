package opencode

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func runOpenCodeCommand(ctx context.Context, args []string, stdoutWriter, stderrWriter io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")

	if stdoutWriter != nil {
		cmd.Stdout = stdoutWriter
	} else {
		cmd.Stdout = os.Stdout
	}
	if stderrWriter != nil {
		cmd.Stderr = stderrWriter
	} else {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode command failed: %w", err)
	}
	return nil
}

func RunCommand(ctx context.Context, model, prompt string, stdoutWriter, stderrWriter io.Writer) error {
	args := []string{"run", "--model", model, prompt}
	return runOpenCodeCommand(ctx, args, stdoutWriter, stderrWriter)
}

func DisplayStats() error {
	return runOpenCodeCommand(context.Background(), []string{"stats"}, os.Stdout, os.Stderr)
}

func runOpenCodeCommandWithRing(ctx context.Context, args []string, ring *RingWriter) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
	cmd.Stdout = io.MultiWriter(os.Stdout, ring)
	cmd.Stderr = io.MultiWriter(os.Stderr, ring)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode execution failed: %w\n\nLast 10 lines of output:\n%s", err, ring.Tail())
	}
	return nil
}

type RingWriter struct {
	n     int
	lines []string
	buf   string
}

func (r *RingWriter) Write(p []byte) (int, error) {
	s := r.buf + string(p)
	parts := strings.Split(s, "\n")
	r.buf = parts[len(parts)-1]
	for _, line := range parts[:len(parts)-1] {
		r.lines = append(r.lines, line)
		if len(r.lines) > r.n {
			r.lines = r.lines[1:]
		}
	}
	return len(p), nil
}

func (r *RingWriter) Tail() string {
	lines := r.lines
	if r.buf != "" {
		lines = append(lines, r.buf)
	}
	return strings.Join(lines, "\n")
}

func NewRingWriter(n int) *RingWriter {
	return &RingWriter{n: n}
}

func RunAgentWithRing(ctx context.Context, model, prompt string, ring *RingWriter) error {
	args := []string{"run", "--model", model, prompt}
	return runOpenCodeCommandWithRing(ctx, args, ring)
}
