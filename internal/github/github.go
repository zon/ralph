package github

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/logger"
)

// ErrNoCommitsBetweenBranches is returned when gh pr create fails because the
// head branch has no commits ahead of the base branch. This is not an error in
// the traditional sense — it means the work was already complete before this
// run started, so there is nothing to open a PR for.
var ErrNoCommitsBetweenBranches = errors.New("no commits between branches")

// IsGHReady checks if the gh CLI is installed and the user is authenticated.
// This consolidates IsGHInstalled and IsGHCLIAvailable into a single function
// with a consistent signature.
func IsGHReady(ctx *context.Context) bool {
	// Check if gh is installed
	cmd := exec.Command("gh", "--version")
	if err := cmd.Run(); err != nil {
		if ctx.IsVerbose() {
			logger.Info("gh CLI is not installed")
		}
		return false
	}

	// Check if authenticated
	cmd = exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		if ctx.IsVerbose() {
			logger.Info("gh CLI is not authenticated")
		}
		return false
	}

	if ctx.IsVerbose() {
		logger.Info("gh CLI is installed and authenticated")
	}

	return true
}

// CreatePR creates a GitHub pull request using gh CLI
// Returns the PR URL on success
func CreatePR(ctx *context.Context, title, body, base, head string) (string, error) {
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
		return handleExistingPR(ctx, err, errOut.String(), out.String(), title, body)
	}

	return parsePRURL(out.String())
}

func handleExistingPR(ctx *context.Context, err error, errStr, outStr, title, body string) (string, error) {
	// GitHub rejects the PR when the head branch has no commits ahead of base.
	// Treat this as a sentinel so callers can decide how to proceed.
	if strings.Contains(errStr, "No commits between") {
		return "", ErrNoCommitsBetweenBranches
	}

	if !strings.Contains(errStr, "a pull request for branch") || !strings.Contains(errStr, "already exists") {
		return "", fmt.Errorf("failed to create PR: %w (output: %s, stderr: %s)", err, outStr, errStr)
	}

	existingURL := extractExistingPRURL(errStr)
	if existingURL == "" {
		return "", fmt.Errorf("failed to create PR: %w (output: %s, stderr: %s)", err, outStr, errStr)
	}

	return updateExistingPR(ctx, existingURL, title, body)
}

func extractExistingPRURL(errStr string) string {
	for _, line := range strings.Split(errStr, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "http") {
			return trimmed
		}
	}
	return ""
}

func updateExistingPR(ctx *context.Context, prURL, title, body string) (string, error) {
	editCmd := exec.Command("gh", "pr", "edit", prURL,
		"--title", title,
		"--body", body,
	)
	var editOut bytes.Buffer
	var editErrOut bytes.Buffer
	editCmd.Stdout = &editOut
	editCmd.Stderr = &editErrOut
	if editErr := editCmd.Run(); editErr != nil {
		return "", fmt.Errorf("failed to update existing PR: %w (output: %s, stderr: %s)", editErr, editOut.String(), editErrOut.String())
	}
	logger.Verbosef("Updated existing PR: %s", prURL)
	return prURL, nil
}

func parsePRURL(output string) (string, error) {
	lines := strings.Split(output, "\n")

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

	logger.Verbosef("Created PR: %s", prURL)
	return prURL, nil
}
