package git

import (
	"fmt"
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

