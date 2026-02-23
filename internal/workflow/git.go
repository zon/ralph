package workflow

import (
	"fmt"
	"os/exec"
	"strings"
)

// getRemoteURL gets the git remote URL.
func getRemoteURL() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// getRepoRoot gets the git repository root directory.
func getRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// getCurrentBranch gets the current git branch name.
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// toHTTPSURL converts a GitHub SSH remote URL to HTTPS.
// SSH format: git@github.com:owner/repo.git -> https://github.com/owner/repo.git
// HTTPS URLs are returned unchanged.
func toHTTPSURL(remoteURL string) string {
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		return "https://github.com/" + strings.TrimPrefix(remoteURL, "git@github.com:")
	}
	return remoteURL
}
