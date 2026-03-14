package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/context"
)

// GetCurrentBranch returns the name of the current git branch
// Returns error if in detached HEAD state
func GetCurrentBranch(ctx *context.Context) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w (output: %s)", err, out.String())
	}

	branch := strings.TrimSpace(out.String())
	if branch == "" {
		return "", fmt.Errorf("failed to determine current branch")
	}

	// Check for detached HEAD state
	if branch == "HEAD" {
		return "", fmt.Errorf("repository is in detached HEAD state, please checkout a branch first")
	}

	return branch, nil
}

// RemoteBranchExists checks if a branch exists on the remote using the already-fetched
// remote-tracking ref. Call Fetch first to ensure refs are up to date.
func RemoteBranchExists(ctx *context.Context, name string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", "origin/"+name)
	return cmd.Run() == nil
}

// CheckoutOrCreateBranch checks out the named branch if it exists on the remote
// (after a prior Fetch), otherwise creates and checks out a new local branch.
func CheckoutOrCreateBranch(ctx *context.Context, name string) error {
	if RemoteBranchExists(ctx, name) {
		if err := CheckoutBranch(ctx, name); err != nil {
			return err
		}
		return nil
	}

	cmd := exec.Command("git", "checkout", "-b", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch '%s': %w (output: %s)", name, err, out.String())
	}
	return nil
}

// CheckoutBranch switches to the specified git branch
func CheckoutBranch(ctx *context.Context, name string) error {
	cmd := exec.Command("git", "checkout", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w (output: %s)", name, err, out.String())
	}

	return nil
}

// HasCommits checks if the current branch has any commits
func HasCommits(ctx *context.Context) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	return cmd.Run() == nil
}

// IsBranchSyncedWithRemote checks if the local branch is in sync with its remote counterpart.
// Returns an error if the remote branch doesn't exist or the local branch is ahead/behind.
func IsBranchSyncedWithRemote(ctx *context.Context, branch string) error {
	// Check that the remote branch exists
	remoteRef := fmt.Sprintf("origin/%s", branch)
	cmd := exec.Command("git", "rev-parse", "--verify", remoteRef)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("branch '%s' has not been pushed to remote - please push before running remotely", branch)
	}

	// Compare local and remote commit hashes
	localCmd := exec.Command("git", "rev-parse", branch)
	var localOut bytes.Buffer
	localCmd.Stdout = &localOut
	if err := localCmd.Run(); err != nil {
		return fmt.Errorf("failed to get local commit for branch '%s': %w", branch, err)
	}

	remoteCmd := exec.Command("git", "rev-parse", remoteRef)
	var remoteOut bytes.Buffer
	remoteCmd.Stdout = &remoteOut
	if err := remoteCmd.Run(); err != nil {
		return fmt.Errorf("failed to get remote commit for branch '%s': %w", branch, err)
	}

	localHash := strings.TrimSpace(localOut.String())
	remoteHash := strings.TrimSpace(remoteOut.String())

	if localHash != remoteHash {
		return fmt.Errorf("branch '%s' is not in sync with remote - please push your changes before running remotely", branch)
	}

	return nil
}

// remoteBranchExists checks whether a branch exists on the remote.
func remoteBranchExists(branch string) bool {
	cmd := exec.Command("git", "ls-remote", "--exit-code", "--heads", "origin", branch)
	return cmd.Run() == nil
}
