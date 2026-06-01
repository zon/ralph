package github

import (
	gocontext "context"
	"fmt"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
)

func CreatePullRequest(ghClient GHClient, ctx *context.Context, proj *project.Project, branchName, baseBranch, prSummary string) (string, error) {
	if ctx.IsWorkflowExecution() {
		owner, repoName := ctx.RepoOwnerAndName()
		if err := ConfigureGitAuth(gocontext.Background(), owner, repoName, DefaultSecretsDir); err != nil {
			return "", fmt.Errorf("failed to refresh GitHub credentials before PR creation: %w", err)
		}
	}

	if !ghClient.IsReady() {
		return "", fmt.Errorf("gh CLI is not ready, please install and authenticate with 'gh auth login'")
	}

	prTitle := proj.Title
	if prTitle == "" {
		prTitle = proj.Slug
	}

	logger.Verbose("Creating GitHub pull request...")
	prURL, err := ghClient.CreatePR(prTitle, prSummary, baseBranch, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to create pull request: %w", err)
	}

	return prURL, nil
}
