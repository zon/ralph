package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/github"
)

type AuthConfig struct {
	Owner string
	Repo  string
}

// ErrWorkflowPermission is returned when a push is rejected because the GitHub App
// token lacks the `workflows` permission required to create or update files under
// .github/workflows/. Retrying will not help — the token must be granted the
// permission before the push can succeed.
var ErrWorkflowPermission = errors.New("push rejected: GitHub App token requires `workflows` permission to push workflow files")

// isWorkflowPermissionError checks whether git push output contains the GitHub
// rejection message for missing `workflows` App permission.
func isWorkflowPermissionError(output string) bool {
	return strings.Contains(output, "refusing to allow a GitHub App to create or update workflow") ||
		strings.Contains(output, "without `workflows` permission")
}

// Fetch fetches from the remote, updating remote-tracking refs.
func Fetch(auth *AuthConfig) error {
	if err := configureAuth(auth); err != nil {
		return fmt.Errorf("failed to configure git auth: %w", err)
	}

	cmd := exec.Command("git", "fetch", "origin")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch from remote: %w (output: %s)", err, out.String())
	}

	return nil
}

// PullRebase pulls remote changes using rebase to avoid merge commits.
// This should be called before pushing to handle cases where the remote
// branch has advanced (e.g. from a previous run or another pod).
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

	cmd := exec.Command("git", "pull", "--rebase", "origin", branch)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull --rebase: %w (output: %s)", err, out.String())
	}

	return nil
}

// Push pushes the current branch or a specified branch to origin.
// If branch is empty, the current branch is pushed.
// Returns the remote URL on success.
func Push(auth *AuthConfig, branch string) (string, error) {
	if err := configureAuth(auth); err != nil {
		return "", fmt.Errorf("failed to configure git auth: %w", err)
	}

	// Determine branch to push
	branchToPush := branch
	if branchToPush == "" {
		var err error
		branchToPush, err = GetCurrentBranch()
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	// Check if there are commits to push
	if !hasCommits() {
		return "", fmt.Errorf("no commits to push on branch '%s'", branchToPush)
	}

	// Push the branch with --set-upstream
	cmd := exec.Command("git", "push", "--set-upstream", "origin", branchToPush)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		if isWorkflowPermissionError(out.String()) {
			return "", fmt.Errorf("%w (output: %s)", ErrWorkflowPermission, out.String())
		}
		return "", fmt.Errorf("failed to push branch '%s': %w (output: %s)", branchToPush, err, out.String())
	}

	// Get the remote URL
	cmd = exec.Command("git", "config", "--get", "remote.origin.url")
	var urlOut bytes.Buffer
	cmd.Stdout = &urlOut
	cmd.Stderr = &urlOut

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w (output: %s)", err, urlOut.String())
	}

	remoteURL := strings.TrimSpace(urlOut.String())

	return remoteURL, nil
}

// PushBranch pushes the specified branch to origin and returns the remote URL
// Deprecated: Use Push instead
func PushBranch(auth *AuthConfig, branch string) (string, error) {
	return Push(auth, branch)
}

// PushCurrentBranch pushes the current branch to origin
// Deprecated: Use Push instead
func PushCurrentBranch(auth *AuthConfig) error {
	_, err := Push(auth, "")
	return err
}

// Clone clones a repository into a directory
func Clone(url, branch, dir string) error {
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, url, dir)

	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository %s: %w (output: %s)", url, err, out.String())
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
