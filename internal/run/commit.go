package run

import (
	"errors"
	"fmt"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// CommitFileChanges stages the given file, commits with the provided message,
// and pushes to the remote branch. It ensures the correct branch is checked out,
// fetches from remote if needed, and pulls rebase before pushing.
func CommitFileChanges(ctx *context.Context, branchName, filePath, commitMsg string) error {
	if err := ensureBranch(ctx, branchName); err != nil {
		return fmt.Errorf("failed to ensure branch %s: %w", branchName, err)
	}

	if err := git.StageFile(filePath); err != nil {
		return fmt.Errorf("failed to stage file %s: %w", filePath, err)
	}

	if err := git.Commit(commitMsg); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	if err := pullAndPush(ctx); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// ensureBranch checks out the given branch, fetching from remote if needed.
// If already on the target branch, it does nothing.
func ensureBranch(ctx *context.Context, branchName string) error {
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	if currentBranch == branchName {
		return nil
	}

	var auth *git.AuthConfig
	if ctx.IsWorkflowExecution() {
		owner, repo := ctx.RepoOwnerAndName()
		auth = &git.AuthConfig{Owner: owner, Repo: repo}
	}

	if err := git.Fetch(auth); err != nil {
		logger.Verbosef("Could not fetch from remote (continuing anyway): %v", err)
	}

	if err := git.CheckoutOrCreateBranch(branchName); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	return nil
}

// pullAndPush pulls remote changes and pushes the current branch
func pullAndPush(ctx *context.Context) error {
	var auth *git.AuthConfig
	if ctx.IsWorkflowExecution() {
		owner, repo := ctx.RepoOwnerAndName()
		auth = &git.AuthConfig{Owner: owner, Repo: repo}
	}

	logger.Verbose("Pulling remote changes before push...")
	if err := git.PullRebase(auth); err != nil {
		return fmt.Errorf("failed to pull before push: %w", err)
	}

	branch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	logger.Verbose("Pushing commit to origin...")
	if _, err := git.Push(auth, branch); err != nil {
		if errors.Is(err, git.ErrWorkflowPermission) {
			return fmt.Errorf("%w: %v", ErrFatalPushError, err)
		}
		return fmt.Errorf("failed to push commit: %w", err)
	}
	logger.Verbose("Pushed commit to origin")

	return nil
}
