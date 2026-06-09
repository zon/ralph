package cmd

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/github"
	workflowtoken "github.com/zon/ralph/internal/orchestration/workflowtoken"
)

// WorkflowTokenCmd generates a GitHub App installation token and configures git auth
type WorkflowTokenCmd struct {
	Owner      string `help:"Repository owner (default: autodetected from git remote)" short:"o"`
	Repo       string `help:"Repository name (default: autodetected from git remote)" short:"r"`
	SecretsDir string `help:"Directory containing GitHub App credentials (default: /secrets/github)" default:"/secrets/github"`
}

// Run executes the workflow token command
func (c *WorkflowTokenCmd) Run() error {
	cmd := &workflowtoken.WorkflowTokenCmd{
		Repo:   &workflowTokenRepoClient{},
		GitHub: &workflowTokenGitHubClient{},
		Git:    &workflowTokenGitClient{},
	}
	flags := workflowtoken.Flags{
		Owner:      c.Owner,
		Repo:       c.Repo,
		SecretsDir: c.SecretsDir,
	}
	return cmd.Run(flags)
}

// workflowTokenRepoClient implements workflowtoken.RepoClient
type workflowTokenRepoClient struct{}

func (c *workflowTokenRepoClient) Resolve(owner, repo string) (string, string, error) {
	if owner == "" || repo == "" {
		detected, err := github.GetRepo(context.Background())
		if err != nil {
			return "", "", fmt.Errorf("failed to autodetect repository from git remote: %w", err)
		}
		if owner == "" {
			owner = detected.Owner
		}
		if repo == "" {
			repo = detected.Name
		}
	}
	if owner == "" {
		return "", "", fmt.Errorf("repository owner is required (use --owner flag or ensure git remote is configured)")
	}
	if repo == "" {
		return "", "", fmt.Errorf("repository name is required (use --repo flag or ensure git remote is configured)")
	}
	return owner, repo, nil
}

// workflowTokenGitHubClient implements workflowtoken.GitHubClient
type workflowTokenGitHubClient struct{}

func (c *workflowTokenGitHubClient) GenerateToken(owner, repo, secretsDir string) (string, error) {
	return github.GenerateInstallationToken(context.Background(), owner, repo, secretsDir)
}

// workflowTokenGitClient implements workflowtoken.GitClient
type workflowTokenGitClient struct{}

func (c *workflowTokenGitClient) ConfigureAuth(token string) error {
	return github.ConfigureTokenAuth(context.Background(), token)
}


