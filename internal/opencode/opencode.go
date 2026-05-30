package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// SessionCollector is a thread-safe collection of session IDs.
type SessionCollector struct {
	mu  sync.Mutex
	ids []string
}

func (c *SessionCollector) Append(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids = append(c.ids, id)
}

func (c *SessionCollector) IDs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]string, len(c.ids))
	copy(result, c.ids)
	return result
}

type contextKey string

const sessionCollectorKey contextKey = "sessionCollector"

func WithSessionCollector(ctx context.Context, c *SessionCollector) context.Context {
	return context.WithValue(ctx, sessionCollectorKey, c)
}

func SessionCollectorFrom(ctx context.Context) *SessionCollector {
	c, _ := ctx.Value(sessionCollectorKey).(*SessionCollector)
	return c
}

// sessionParser wraps an io.Writer and extracts session ID from JSON lines.
type sessionParser struct {
	w         io.Writer
	sessionID string
	mu        sync.Mutex
}

func newSessionParser(w io.Writer) *sessionParser {
	return &sessionParser{w: w}
}

func (s *sessionParser) Write(p []byte) (int, error) {
	s.mu.Lock()
	if s.sessionID == "" {
		s.parseSessionID(p)
	}
	s.mu.Unlock()

	if s.w != nil {
		return s.w.Write(p)
	}
	return len(p), nil
}

func (s *sessionParser) parseSessionID(p []byte) {
	var obj struct {
		SessionID string `json:"sessionID"`
	}
	if err := json.Unmarshal(p, &obj); err == nil && obj.SessionID != "" {
		s.sessionID = obj.SessionID
		return
	}
	for _, line := range bytes.Split(p, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		if err := json.Unmarshal(line, &obj); err == nil && obj.SessionID != "" {
			s.sessionID = obj.SessionID
			return
		}
	}
}

func runOpenCodeCommand(ctx context.Context, args []string, stdoutWriter, stderrWriter io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")

	parser := newSessionParser(stdoutWriter)
	cmd.Stdout = parser

	if stderrWriter != nil {
		cmd.Stderr = stderrWriter
	} else {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode command failed: %w", err)
	}

	if collector := SessionCollectorFrom(ctx); collector != nil && parser.sessionID != "" {
		collector.Append(parser.sessionID)
	}
	return nil
}

func RunCommand(ctx context.Context, model, variant, prompt string, stdoutWriter, stderrWriter io.Writer) error {
	args := []string{"run", "--format", "json", "--model", model}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	args = append(args, prompt)
	return runOpenCodeCommand(ctx, args, stdoutWriter, stderrWriter)
}

// SessionStats holds cost and token counts for a single session.
type SessionStats struct {
	InputTokens  int64
	OutputTokens int64
	Cost         float64
}

// ExportSession runs opencode export <sessionID>, parses the JSON response,
// and populates SessionStats from the session's cost and token fields.
func ExportSession(sessionID string) (SessionStats, error) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "opencode", "export", sessionID)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return SessionStats{}, fmt.Errorf("opencode export failed: %w\nstderr: %s", err, stderr.String())
	}

	var resp struct {
		Info struct {
			Cost   float64 `json:"cost"`
			Tokens struct {
				Input  int `json:"input"`
				Output int `json:"output"`
			} `json:"tokens"`
		} `json:"info"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return SessionStats{}, fmt.Errorf("failed to parse opencode export JSON: %w", err)
	}

	return SessionStats{
		InputTokens:  int64(resp.Info.Tokens.Input),
		OutputTokens: int64(resp.Info.Tokens.Output),
		Cost:         resp.Info.Cost,
	}, nil
}


func runOpenCodeCommandWithRing(ctx context.Context, args []string, ring *RingWriter) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
	parser := newSessionParser(os.Stdout)
	cmd.Stdout = io.MultiWriter(parser, ring)
	cmd.Stderr = io.MultiWriter(os.Stderr, ring)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode execution failed: %w\n\nLast 10 lines of output:\n%s", err, ring.Tail())
	}

	if collector := SessionCollectorFrom(ctx); collector != nil && parser.sessionID != "" {
		collector.Append(parser.sessionID)
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

func RunAgentWithRing(ctx context.Context, model, variant, prompt string, ring *RingWriter) error {
	args := []string{"run", "--format", "json", "--model", model}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	args = append(args, prompt)
	return runOpenCodeCommandWithRing(ctx, args, ring)
}
