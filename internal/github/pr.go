package github

import (
	"fmt"
	"strings"

	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

func CreatePullRequest(out *output.Client, ghClient GHClient, proj *project.Project, branchName, baseBranch, prSummary string) (string, error) {
	if !ghClient.IsReady() {
		return "", fmt.Errorf("gh CLI is not ready, please install and authenticate with 'gh auth login'")
	}

	prTitle, prBody := splitTitle(prSummary)
	if prTitle == "" {
		prTitle = proj.Title
	}
	if prTitle == "" {
		prTitle = proj.Slug
	}
	if prBody == "" {
		prBody = prSummary
	}

	out.Debug("Creating GitHub pull request...")
	prURL, err := ghClient.CreatePR(prTitle, prBody, baseBranch, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to create pull request: %w", err)
	}

	return prURL, nil
}

// splitTitle separates the H1 title that starts an AI-generated PR summary
// from the body that follows it. A summary that does not begin with an H1
// heading yields an empty title and the full summary as the body.
func splitTitle(summary string) (title, body string) {
	content := strings.TrimLeft(summary, "\n")
	first := content
	rest := ""
	if newline := strings.IndexByte(content, '\n'); newline >= 0 {
		first = content[:newline]
		rest = strings.TrimLeft(content[newline+1:], "\n")
	}
	if !strings.HasPrefix(strings.TrimSpace(first), "# ") {
		return "", summary
	}
	title = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(first), "# "))
	if title == "" {
		return "", summary
	}
	return title, rest
}
