package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// Installed reports whether the opencode CLI is on the PATH.
func Installed() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
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

// runArgs builds the argument list for an `opencode run` invocation.
//
// opencode 2 removed the --dir and --variant options from `run`. The project
// directory now comes from the process working directory, so ralph relies on
// os.Chdir to place the process inside the run's worktree, and the reasoning
// variant is encoded in the model reference as provider/model#variant.
//
// --standalone runs the prompt against a private server instead of opencode's
// shared background service. Ralph drains the process group after each run
// (see execOpenCode), so the server must be a child of the CLI rather than a
// detached daemon.
//
// When no model is configured, the --model option is omitted entirely so
// opencode falls back to its own configured default model. A variant cannot be
// requested on its own: opencode 2 has no standalone variant option, so it is
// an error to ask for a variant without a model.
func runArgs(model, variant, agent, prompt string) ([]string, error) {
	args := []string{"run", "--standalone"}
	if model != "" {
		ref := model
		if variant != "" && !strings.Contains(ref, "#") {
			ref = ref + "#" + variant
		}
		args = append(args, "--model", ref)
	} else if variant != "" {
		return nil, fmt.Errorf("variant %q requires a model: opencode 2 encodes the variant in the model reference (provider/model#variant)", variant)
	}
	if agent != "" {
		args = append(args, "--agent", agent)
	}
	args = append(args, prompt)
	return args, nil
}

func (c *Client) RunCommand(ctx context.Context, model, variant, agent, prompt string, stdoutWriter, stderrWriter io.Writer) error {
	args, err := runArgs(model, variant, agent, prompt)
	if err != nil {
		return err
	}
	if stdoutWriter == nil {
		stdoutWriter = os.Stdout
	}
	if stderrWriter == nil {
		stderrWriter = os.Stderr
	}
	return execOpenCode(ctx, args, stdoutWriter, stderrWriter)
}

func (c *Client) RunAgent(ctx context.Context, model, variant, agent, prompt string) error {
	args, err := runArgs(model, variant, agent, prompt)
	if err != nil {
		return err
	}
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
	err := execOpenCode(context.Background(), []string{"stats", "--standalone", "--json"}, &stdout, io.Discard)
	if err != nil {
		return Stats{}, err
	}
	return parseStatsJSON(stdout.Bytes())
}

func (c *Client) DisplayStats() error {
	return execOpenCode(context.Background(), []string{"stats", "--standalone"}, os.Stdout, os.Stderr)
}

// statsJSON mirrors the subset of `opencode stats --json` that ralph reports.
// opencode 2 emits machine-readable JSON (tokens.input, tokens.output, cost)
// instead of the box-drawing table opencode 1 printed.
type statsJSON struct {
	Tokens struct {
		Input  int64 `json:"input"`
		Output int64 `json:"output"`
	} `json:"tokens"`
	Cost float64 `json:"cost"`
}

func parseStatsJSON(data []byte) (Stats, error) {
	var payload statsJSON
	if err := json.Unmarshal(data, &payload); err != nil {
		return Stats{}, fmt.Errorf("failed to parse stats output: %w", err)
	}
	return Stats{
		InputTokens:  payload.Tokens.Input,
		OutputTokens: payload.Tokens.Output,
		Cost:         payload.Cost,
	}, nil
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
