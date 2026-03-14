package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/context"
)

// Merge merges a branch into the current branch
func Merge(ctx *context.Context, branch string) error {
	cmd := exec.Command("git", "merge", branch, "--no-edit")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("merge failed: %w (output: %s)", err, out.String())
	}
	return nil
}

// AbortMerge aborts an in-progress merge
func AbortMerge(ctx *context.Context) error {
	cmd := exec.Command("git", "merge", "--abort")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to abort merge: %w (output: %s)", err, out.String())
	}
	return nil
}

// MergeBase returns the merge base of two commits
func MergeBase(ctx *context.Context, a, b string) (string, error) {
	cmd := exec.Command("git", "merge-base", a, b)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to find merge base: %w (output: %s)", err, out.String())
	}
	return strings.TrimSpace(out.String()), nil
}
