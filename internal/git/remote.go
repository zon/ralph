package git

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zon/ralph/internal/github"
)

type AuthConfig struct {
	Owner string
	Repo  string
}

var ErrWorkflowPermission = errors.New("push rejected: GitHub App token requires `workflows` permission to push workflow files")

func isWorkflowPermissionError(output string) bool {
	return strings.Contains(output, "refusing to allow a GitHub App to create or update workflow") ||
		strings.Contains(output, "without `workflows` permission")
}

// Fetch fetches from the remote, updating remote-tracking refs.
func Fetch(auth *AuthConfig) error {
	if err := configureAuth(auth); err != nil {
		return fmt.Errorf("failed to configure git auth: %w", err)
	}

	_, err := runGit("fetch", "origin")
	if err != nil {
		return fmt.Errorf("failed to fetch from remote: %w", err)
	}

	return nil
}

// PullRebase pulls remote changes using rebase to avoid merge commits.
func PullRebase(auth *AuthConfig) error {
	if err := configureAuth(auth); err != nil {
		return fmt.Errorf("failed to configure git auth: %w", err)
	}

	branch, err := GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch for pull: %w", err)
	}

	if !remoteBranchExists(branch) {
		return nil
	}

	_, err = runGit("pull", "--rebase", "origin", branch)
	if err != nil {
		return fmt.Errorf("failed to pull --rebase: %w", err)
	}

	return nil
}

// RemoteURL returns the URL of the origin remote.
func RemoteURL() (string, error) {
	remoteURL, err := runGit("config", "--get", "remote.origin.url")
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return remoteURL, nil
}

// Push pushes the current branch or a specified branch to origin.
func Push(auth *AuthConfig, branch string) (string, error) {
	if err := configureAuth(auth); err != nil {
		return "", fmt.Errorf("failed to configure git auth: %w", err)
	}

	branchToPush := branch
	if branchToPush == "" {
		var err error
		branchToPush, err = GetCurrentBranch()
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	if !hasCommits() {
		return "", fmt.Errorf("no commits to push on branch '%s'", branchToPush)
	}

	output, err := runGit("push", "--set-upstream", "origin", branchToPush)
	if err != nil {
		if isWorkflowPermissionError(output) {
			return "", fmt.Errorf("%w (output: %s)", ErrWorkflowPermission, output)
		}
		return "", fmt.Errorf("failed to push branch '%s': %w", branchToPush, err)
	}

	return RemoteURL()
}

// Clone clones a repository into a directory
func Clone(url, branch, dir string) error {
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, url, dir)

	_, err := runGit(args...)
	if err != nil {
		return fmt.Errorf("failed to clone repository %s: %w", url, err)
	}
	return nil
}

// configureAuth refreshes the GitHub App token and configures git HTTPS auth.
func configureAuth(auth *AuthConfig) error {
	if auth == nil {
		return nil
	}
	return github.ConfigureGitAuth(context.Background(), auth.Owner, auth.Repo, github.DefaultSecretsDir)
}
