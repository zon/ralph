package github

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/logger"
)

// ErrNoCommitsBetweenBranches is returned when gh pr create fails because the
// head branch has no commits ahead of the base branch. This is not an error in
// the traditional sense — it means the work was already complete before this
// run started, so there is nothing to open a PR for.
var ErrNoCommitsBetweenBranches = errors.New("no commits between branches")

// IsReady checks if the gh CLI is installed and the user is authenticated.
// This consolidates IsGHInstalled and IsGHCLIAvailable into a single function
// with a consistent signature.
func IsReady() bool {
	// Check if gh is installed
	cmd := exec.Command("gh", "--version")
	if err := cmd.Run(); err != nil {
		return false
	}

	// Check if authenticated
	cmd = exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

// FindExistingPR checks if an open PR already exists for the given head branch.
// Returns the PR URL if found, or empty string if no PR exists.
func FindExistingPR(head string) (string, error) {
	cmd := exec.Command("gh", "pr", "list",
		"--head", head,
		"--state", "open",
		"--json", "url",
		"--limit", "1",
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to check for existing PRs: %w", err)
	}

	output := out.String()
	if !strings.Contains(output, "url") {
		return "", nil
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "http") {
			return trimmed, nil
		}
	}

	return "", nil
}

// CreatePR creates a GitHub pull request using gh CLI.
// First checks if an open PR already exists for the branch.
// If found, updates the existing PR's title and body (preserving base branch).
// If not found, creates a new PR.
func CreatePR(title, body, base, head string) (string, error) {
	existingPR, err := FindExistingPR(head)
	if err != nil {
		return "", err
	}

	if existingPR != "" {
		return updateExistingPR(existingPR, title, body)
	}

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

	if createErr := cmd.Run(); createErr != nil {
		return handleExistingPR(createErr, errOut.String(), out.String(), title, body)
	}

	return parsePRURL(out.String())
}

func handleExistingPR(err error, errStr, outStr, title, body string) (string, error) {
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

	return updateExistingPR(existingURL, title, body)
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

func updateExistingPR(prURL, title, body string) (string, error) {
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
