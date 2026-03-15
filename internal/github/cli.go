package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GetRepo extracts the repository owner and name from git remote origin.
func GetRepo(ctx context.Context) (Repo, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "--get", "remote.origin.url")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Repo{}, fmt.Errorf("failed to get remote.origin.url: %w (stderr: %s)", err, stderr.String())
	}

	return ParseRemoteURL(strings.TrimSpace(stdout.String()))
}

// ParseRemoteURL parses a GitHub remote URL and returns the repository.
// Supported formats:
//
//	git@github.com:owner/repo.git
//	https://github.com/owner/repo.git
//	https://github.com/owner/repo
func ParseRemoteURL(remoteURL string) (Repo, error) {
	if remoteURL == "" {
		return Repo{}, fmt.Errorf("remote.origin.url is empty")
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
		return Repo{}, fmt.Errorf("not a GitHub repository URL: %s", remoteURL)
	}

	repoPath = strings.TrimSuffix(repoPath, ".git")

	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return Repo{}, fmt.Errorf("invalid repository path: %s", repoPath)
	}

	return MakeRepo(parts[0], parts[1]), nil
}
