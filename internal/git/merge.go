package git

import (
	"fmt"
	"strings"
)

// Merge merges a branch into the current branch
func Merge(branch string) error {
	_, err := runGit("merge", branch, "--no-edit")
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}
	return nil
}

// AbortMerge aborts an in-progress merge
func AbortMerge() error {
	_, err := runGit("merge", "--abort")
	if err != nil {
		return fmt.Errorf("failed to abort merge: %w", err)
	}
	return nil
}

// MergeBase returns the merge base of two commits
func MergeBase(a, b string) (string, error) {
	out, err := runGit("merge-base", a, b)
	if err != nil {
		return "", fmt.Errorf("failed to find merge base: %w", err)
	}
	return strings.TrimSpace(out), nil
}
