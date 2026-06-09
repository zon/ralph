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

// NeedsMerge checks whether the current HEAD needs to merge the given branch.
// Returns true if the branch exists locally and is ahead of the merge base
// with HEAD.
func NeedsMerge(branch string) (bool, error) {
	_, err := runGit("rev-parse", "--verify", branch)
	if err != nil {
		return false, nil
	}
	mergeBase, err := runGit("merge-base", "HEAD", branch)
	if err != nil {
		return false, fmt.Errorf("failed to find merge base: %w", err)
	}
	baseCommit, err := runGit("rev-parse", branch)
	if err != nil {
		return false, fmt.Errorf("failed to get base commit: %w", err)
	}
	return strings.TrimSpace(mergeBase) != strings.TrimSpace(baseCommit), nil
}

