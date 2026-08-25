package opencode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// opencodeProcessGroupWaitTimeout bounds how long execOpenCode waits for
// opencode's process group to drain after the CLI exits. The CLI can exit
// while a process it spawned (a sub-agent, the local server) is still writing,
// so ralph drains the group before treating the run as complete. The wait is
// best-effort: a lingering group is reported, not fatal.
var opencodeProcessGroupWaitTimeout = 30 * time.Second

const opencodeProcessGroupPollInterval = 100 * time.Millisecond

type OCClient interface {
	RunCommand(ctx context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error
	RunAgent(ctx context.Context, model, variant, agent, prompt string) error
	GetStats() (Stats, error)
	DisplayStats() error
}

type Client struct{}

func New() *Client {
	return &Client{}
}

func execOpenCode(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	// Run opencode in its own process group so waitForGroupExit can wait for
	// every process it spawns to exit before the run is reported complete.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("opencode command failed: %w", err)
	}
	if err := waitForGroupExit(cmd.Process.Pid); err != nil {
		// A lingering process (e.g. a server) must not fail the run; the
		// pull-before-push step tolerates any leftover writes.
		if stderr != nil {
			fmt.Fprintf(stderr, "opencode: %v\n", err)
		}
	}
	return nil
}

// waitForGroupExit blocks until every process in the group led by pid has
// exited or opencodeProcessGroupWaitTimeout elapses. pid is the group leader:
// Setpgid makes the child its own group leader, so the process group id equals
// the child's pid.
func waitForGroupExit(pid int) error {
	deadline := time.Now().Add(opencodeProcessGroupWaitTimeout)
	for {
		err := syscall.Kill(-pid, 0)
		switch {
		case err == nil:
			// The group still has members.
		case errors.Is(err, syscall.ESRCH):
			return nil
		case errors.Is(err, syscall.EPERM):
			// A member exists that we cannot signal; keep waiting.
		default:
			return fmt.Errorf("failed to check process group %d: %w", pid, err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("process group %d still active after %s", pid, opencodeProcessGroupWaitTimeout)
		}
		time.Sleep(opencodeProcessGroupPollInterval)
	}
}

func (c *Client) RunCommand(ctx context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
	args := []string{"run", "--model", model}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	if agent != "" {
		args = append(args, "--agent", agent)
	}
	args = append(args, prompt)
	if stdoutWriter == nil {
		stdoutWriter = os.Stdout
	}
	if stderrWriter == nil {
		stderrWriter = os.Stderr
	}
	return execOpenCode(ctx, args, stdoutWriter, stderrWriter)
}

func (c *Client) RunAgent(ctx context.Context, model, variant, agent, prompt string) error {
	args := []string{"run", "--model", model}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	if agent != "" {
		args = append(args, "--agent", agent)
	}
	args = append(args, prompt)
	ring := &ringWriter{n: 10}
	stdout := io.MultiWriter(os.Stdout, ring)
	stderr := io.MultiWriter(os.Stderr, ring)
	if err := execOpenCode(ctx, args, stdout, stderr); err != nil {
		return fmt.Errorf("opencode execution failed: %w\n\nLast 10 lines of output:\n%s", err, ring.Tail())
	}
	return nil
}

type Stats struct {
	InputTokens  int64
	OutputTokens int64
	Cost         float64
}

// Formatted returns a one-line human-readable summary of the token usage and
// cost.
func (s Stats) Formatted() string {
	return fmt.Sprintf("Input tokens: %s, Output tokens: %s, Cost: $%.2f", formatTokens(s.InputTokens), formatTokens(s.OutputTokens), s.Cost)
}

func formatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func (c *Client) GetStats() (Stats, error) {
	var stdout bytes.Buffer
	err := execOpenCode(context.Background(), []string{"stats"}, &stdout, io.Discard)
	if err != nil {
		return Stats{}, err
	}
	return parseStatsOutput(stdout.String())
}

func (c *Client) DisplayStats() error {
	return execOpenCode(context.Background(), []string{"stats"}, os.Stdout, os.Stderr)
}

func parseStatsOutput(output string) (Stats, error) {
	var stats Stats
	var foundInput, foundOutput, foundCost bool
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		var value string
		if strings.Contains(line, "Input") {
			value = extractValue(line)
			if value == "" {
				continue
			}
			tokens, err := parseTokenValue(value)
			if err != nil {
				return Stats{}, fmt.Errorf("failed to parse input tokens from %q: %w", value, err)
			}
			stats.InputTokens = tokens
			foundInput = true
		} else if strings.Contains(line, "Output") {
			value = extractValue(line)
			if value == "" {
				continue
			}
			tokens, err := parseTokenValue(value)
			if err != nil {
				return Stats{}, fmt.Errorf("failed to parse output tokens from %q: %w", value, err)
			}
			stats.OutputTokens = tokens
			foundOutput = true
		} else if strings.Contains(line, "Total Cost") {
			value = extractValue(line)
			if value == "" {
				continue
			}
			cost, err := parseCostValue(value)
			if err != nil {
				return Stats{}, fmt.Errorf("failed to parse cost from %q: %w", value, err)
			}
			stats.Cost = cost
			foundCost = true
		}
	}

	if !foundInput || !foundOutput || !foundCost {
		return Stats{}, fmt.Errorf("failed to parse stats output: missing required fields (input=%v, output=%v, cost=%v)", foundInput, foundOutput, foundCost)
	}

	return stats, nil
}

func extractValue(line string) string {
	line = strings.TrimRight(line, " ")
	sep := "│"
	if idx := strings.LastIndex(line, sep); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func parseTokenValue(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty token value")
	}
	var multiplier int64 = 1
	last := s[len(s)-1]
	switch last {
	case 'K', 'k':
		multiplier = 1000
		s = s[:len(s)-1]
	case 'M', 'm':
		multiplier = 1000000
		s = s[:len(s)-1]
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid token value %q: %w", s, err)
	}
	return int64(val * float64(multiplier)), nil
}

func parseCostValue(s string) (float64, error) {
	s = strings.TrimPrefix(s, "$")
	if s == "" {
		return 0, fmt.Errorf("empty cost value")
	}
	return strconv.ParseFloat(s, 64)
}

type ringWriter struct {
	n     int
	lines []string
	buf   string
}

func (r *ringWriter) Write(p []byte) (int, error) {
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

func (r *ringWriter) Tail() string {
	lines := r.lines
	if r.buf != "" {
		lines = append(lines, r.buf)
	}
	return strings.Join(lines, "\n")
}

var fatalOpenCodePatterns = []string{
	"Insufficient Balance",
	"insufficient balance",
	"billing",
	"account",
	"payment required",
	"quota exceeded",
}

func IsFatalError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	for _, pattern := range fatalOpenCodePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}
