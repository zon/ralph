package github

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/output"
)

func handleExistingPR(out *output.Client, err error, errStr, outStr, title, body string) (string, error) {
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

	return updateExistingPR(out, existingURL, title, body)
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

func updateExistingPR(out *output.Client, prURL, title, body string) (string, error) {
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
	out.Debugf("Updated existing PR: %s", prURL)
	return prURL, nil
}

func parsePRURL(out *output.Client, output string) (string, error) {
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

	out.Debugf("Created PR: %s", prURL)
	return prURL, nil
}


