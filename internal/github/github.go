package github

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/logger"
)

// IsGHInstalled checks if the gh CLI is installed
func IsGHInstalled(ctx *context.Context) bool {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would check if gh CLI is installed")
		return true
	}

	cmd := exec.Command("gh", "--version")
	err := cmd.Run()
	installed := err == nil

	if ctx.IsVerbose() {
		if installed {
			logger.Info("gh CLI is installed")
		} else {
			logger.Info("gh CLI is not installed")
		}
	}

	return installed
}

// IsAuthenticated checks if the user is authenticated with GitHub via gh CLI
func IsAuthenticated(ctx *context.Context) bool {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would check if gh CLI is authenticated")
		return true
	}

	cmd := exec.Command("gh", "auth", "status")
	err := cmd.Run()
	authenticated := err == nil

	if ctx.IsVerbose() {
		if authenticated {
			logger.Info("gh CLI is authenticated")
		} else {
			logger.Info("gh CLI is not authenticated")
		}
	}

	return authenticated
}

// CreatePR creates a GitHub pull request using gh CLI
// Returns the PR URL on success
func CreatePR(ctx *context.Context, title, body, base, head string) (string, error) {
	if ctx.IsDryRun() {
		logger.Info("[DRY-RUN] Would create PR:")
		logger.Info("[DRY-RUN]   Title: %s", title)
		logger.Info("[DRY-RUN]   Base: %s", base)
		logger.Info("[DRY-RUN]   Head: %s", head)
		logger.Info("[DRY-RUN]   Body: %s", truncate(body, 200))
		return "https://github.com/dry-run/repo/pull/123", nil
	}

	// Create PR using gh pr create
	cmd := exec.Command("gh", "pr", "create",
		"--title", title,
		"--body", body,
		"--base", base,
		"--head", head,
	)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create PR: %w (output: %s, stderr: %s)", err, out.String(), errOut.String())
	}

	// Parse PR URL from output
	output := strings.TrimSpace(out.String())
	lines := strings.Split(output, "\n")

	// gh pr create typically outputs the URL on the last line
	prURL := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "http") {
			prURL = trimmed
		}
	}

	if prURL == "" {
		return "", fmt.Errorf("failed to parse PR URL from gh output: %s", output)
	}

	if ctx.IsVerbose() {
		logger.Info("Created PR: %s", prURL)
	}

	logger.Success("Pull request created: %s", prURL)

	return prURL, nil
}

// truncate truncates a string to maxLen characters with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
