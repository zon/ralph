package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

type GitAuthConfigurator interface {
	ConfigureGitAuth(ctx context.Context, owner, repo, secretsDir string) error
	DefaultSecretsDir() string
}

var authConfigurator GitAuthConfigurator

func SetAuthConfigurator(ac GitAuthConfigurator) {
	authConfigurator = ac
}

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

func RemoteURL() (string, error) {
	remoteURL, err := runGit("config", "--get", "remote.origin.url")
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return remoteURL, nil
}

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

func configureAuth(auth *AuthConfig) error {
	if auth == nil || authConfigurator == nil {
		return nil
	}
	return authConfigurator.ConfigureGitAuth(context.Background(), auth.Owner, auth.Repo, authConfigurator.DefaultSecretsDir())
}

var ErrFatalPushError = errors.New("fatal push error")

func PullAndPush(isWorkflow bool, owner, repo string) error {
	var auth *AuthConfig
	if isWorkflow {
		auth = &AuthConfig{Owner: owner, Repo: repo}
	}

	if err := PullRebase(auth); err != nil {
		return fmt.Errorf("failed to pull before push: %w", err)
	}

	branch, err := GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if _, err := Push(auth, branch); err != nil {
		return err
	}

	return nil
}
