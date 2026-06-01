package github

import (
	"fmt"

	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
)

func CreatePullRequest(ghClient GHClient, proj *project.Project, branchName, baseBranch, prSummary string) (string, error) {
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
