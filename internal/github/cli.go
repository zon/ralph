package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// IsGHCLIAvailable checks if the gh CLI is installed and the user is authenticated.
func IsGHCLIAvailable(ctx context.Context) bool {
	_, err := exec.LookPath("gh")
	if err != nil {
		return false
	}

	cmd := exec.CommandContext(ctx, "gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

// GetAuthenticatedUser returns the login of the currently authenticated gh user.
func GetAuthenticatedUser(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "api", "user", "--jq", ".login")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w (stderr: %s)", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// SwitchUser switches the active gh account to the given username.
func SwitchUser(ctx context.Context, username string) error {
	cmd := exec.CommandContext(ctx, "gh", "auth", "switch", "--user", username)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to GitHub user %q: %w (stderr: %s)", username, err, stderr.String())
	}

	return nil
}

// FindSSHKey searches for an SSH key by title on GitHub.
// Returns the key ID if found, empty string if not found.
func FindSSHKey(ctx context.Context, title string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "ssh-key", "list")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to list SSH keys: %w (stderr: %s)", err, stderr.String())
	}

	// Parse output to find key with matching title.
	// Format: "TITLE KEY_TYPE KEY_DATA CREATED_DATE KEY_ID TYPE"
	// Example: "ralph-myrepo ssh-ed25519 AAAAC3... 2025-02-15T12:00:00Z 123456789 authentication"
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "warning:") || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		if fields[0] == title {
			// Key ID is the second-to-last field.
			return fields[len(fields)-2], nil
		}
	}

	return "", nil
}

// DeleteSSHKey deletes an SSH key from GitHub by its ID.
func DeleteSSHKey(ctx context.Context, keyID string) error {
	cmd := exec.CommandContext(ctx, "gh", "ssh-key", "delete", keyID, "--yes")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete SSH key: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// GetRepo extracts the repository name and owner from git remote origin.
// Returns: repoName, repoOwner, error
func GetRepo(ctx context.Context) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "--get", "remote.origin.url")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("failed to get remote.origin.url: %w (stderr: %s)", err, stderr.String())
	}

	return parseGitHubRemoteURL(strings.TrimSpace(stdout.String()))
}

// parseGitHubRemoteURL parses a GitHub remote URL and returns the repo name and owner.
// Supported formats:
//
//	git@github.com:owner/repo.git
//	https://github.com/owner/repo.git
//	https://github.com/owner/repo
func parseGitHubRemoteURL(remoteURL string) (string, string, error) {
	if remoteURL == "" {
		return "", "", fmt.Errorf("remote.origin.url is empty")
	}

	var repoPath string

	if strings.HasPrefix(remoteURL, "git@github.com:") {
		repoPath = strings.TrimPrefix(remoteURL, "git@github.com:")
	} else if strings.Contains(remoteURL, "github.com/") {
		parts := strings.Split(remoteURL, "github.com/")
		if len(parts) > 1 {
			repoPath = parts[1]
		}
	} else {
		return "", "", fmt.Errorf("not a GitHub repository URL: %s", remoteURL)
	}

	repoPath = strings.TrimSuffix(repoPath, ".git")

	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository path: %s", repoPath)
	}

	return parts[1], parts[0], nil
}
