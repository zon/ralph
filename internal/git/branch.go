package git

import (
	"fmt"
	"strings"
)

// GetCurrentBranch returns the name of the current git branch
// Returns error if in detached HEAD state
func GetCurrentBranch() (string, error) {
	branch, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	if branch == "" {
		return "", fmt.Errorf("failed to determine current branch")
	}

	if branch == "HEAD" {
		return "", fmt.Errorf("repository is in detached HEAD state, please checkout a branch first")
	}

	return branch, nil
}

// CheckoutOrCreateBranch checks out the named branch if it exists on the remote
// (after a prior Fetch), otherwise creates and checks out a new local branch.
func CheckoutOrCreateBranch(name string) error {
	if remoteBranchExists(name) {
		if err := checkoutBranch(name); err != nil {
			return err
		}
		return nil
	}

	_, err := runGit("checkout", "-b", name)
	if err != nil {
		return fmt.Errorf("failed to create branch '%s': %w", name, err)
	}
	return nil
}

// checkoutBranch switches to the specified git branch
func checkoutBranch(name string) error {
	_, err := runGit("checkout", name)
	if err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w", name, err)
	}
	return nil
}

// hasCommits checks if the current branch has any commits
func hasCommits() bool {
	_, err := runGit("rev-parse", "--verify", "HEAD")
	return err == nil
}

// IsBranchSyncedWithRemote checks if the local branch is in sync with its remote counterpart.
// Returns an error if the remote branch doesn't exist or the local branch is ahead/behind.
func IsBranchSyncedWithRemote(branch string) error {
	remoteRef := fmt.Sprintf("origin/%s", branch)
	_, err := runGit("rev-parse", "--verify", remoteRef)
	if err != nil {
		return fmt.Errorf("branch '%s' has not been pushed to remote - please push before running remotely", branch)
	}

	localHash, err := runGit("rev-parse", branch)
	if err != nil {
		return fmt.Errorf("failed to get local commit for branch '%s': %w", branch, err)
	}

	remoteHash, err := runGit("rev-parse", remoteRef)
	if err != nil {
		return fmt.Errorf("failed to get remote commit for branch '%s': %w", branch, err)
	}

	if strings.TrimSpace(localHash) != strings.TrimSpace(remoteHash) {
		return fmt.Errorf("branch '%s' is not in sync with remote - please push your changes before running remotely", branch)
	}

	return nil
}

// remoteBranchExists checks whether a branch exists on the remote.
func remoteBranchExists(branch string) bool {
	_, err := runGit("ls-remote", "--exit-code", "--heads", "origin", branch)
	return err == nil
}
