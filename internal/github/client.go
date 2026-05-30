package github

import (
	"errors"
	"fmt"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
)

type Client struct {
	ctx        *context.Context
	baseBranch string
}

func NewClient(ctx *context.Context, baseBranch string) *Client {
	return &Client{
		ctx:        ctx,
		baseBranch: baseBranch,
	}
}

func (a *Client) CreatePR(proj *project.Project) error {
	commitLog, err := git.GetCommitLog(a.baseBranch, 100)
	if err != nil {
		return fmt.Errorf("failed to get commit log: %w", err)
	}

	allComplete, passingCount, failingCount := project.CheckCompletion(proj)
	projectStatus := fmt.Sprintf("%d passing, %d failing (complete: %v)", passingCount, failingCount, allComplete)

	prSummary, err := ai.GeneratePRSummary(a.ctx, proj.Title, projectStatus, a.baseBranch, commitLog)
	if err != nil {
		return fmt.Errorf("failed to generate PR summary: %w", err)
	}

	branchName := git.SanitizeBranchName(proj.Slug)

	prURL, err := CreatePullRequest(a.ctx, proj, branchName, a.baseBranch, prSummary)
	if err != nil {
		if errors.Is(err, ErrNoCommitsBetweenBranches) {
			logger.Verbose("No commits ahead of base branch — all requirements were already passing; skipping PR creation")
			return nil
		}
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	logger.Info(prURL)
	return nil
}
