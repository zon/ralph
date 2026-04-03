package run

import (
	"fmt"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
)

// switchToBranchIfNeeded switches to branchName if not already on it, fetching remote changes if needed.
// Returns auth config if running in workflow execution.
func switchToBranchIfNeeded(ctx *context.Context, branchName string) (*git.AuthConfig, error) {
	var auth *git.AuthConfig
	if ctx.IsWorkflowExecution() {
		owner, repo := ctx.RepoOwnerAndName()
		auth = &git.AuthConfig{Owner: owner, Repo: repo}
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	if currentBranch != branchName {
		if err := git.Fetch(auth); err != nil {
			logger.Verbosef("Could not fetch from remote (continuing anyway): %v", err)
		}
		if err := git.CheckoutOrCreateBranch(branchName); err != nil {
			return nil, fmt.Errorf("failed to checkout review branch: %w", err)
		}
	}
	return auth, nil
}

// CommitFileAndPush stages a single file, commits with the given message,
// switches to the specified branch if not already on it, and pushes the commit.
// If the branch does not exist, it will be created.
// This is intended for review workflow where each finding is committed separately.
func CommitFileAndPush(ctx *context.Context, filePath, branchName, commitMsg string) error {
	if _, err := switchToBranchIfNeeded(ctx, branchName); err != nil {
		return err
	}

	if err := git.StageFile(filePath); err != nil {
		return fmt.Errorf("failed to stage project file: %w", err)
	}

	if err := git.Commit(commitMsg); err != nil {
		return fmt.Errorf("failed to commit review findings: %w", err)
	}

	if err := PullAndPush(ctx); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}
