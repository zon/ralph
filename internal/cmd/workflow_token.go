package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/git"
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
	appIDPath := filepath.Join(secretsDir, "app-id")
	appIDBytes, err := os.ReadFile(appIDPath)
	if err != nil {
		return "", fmt.Errorf("failed to read app ID from %s: %w", appIDPath, err)
	}
	appID := strings.TrimSpace(string(appIDBytes))
	if appID == "" {
		return "", fmt.Errorf("app ID is empty in %s", appIDPath)
	}

	privateKeyPath := filepath.Join(secretsDir, "private-key")
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key from %s: %w", privateKeyPath, err)
	}
	if len(privateKeyBytes) == 0 {
		return "", fmt.Errorf("private key is empty in %s", privateKeyPath)
	}

	jwtToken, err := github.GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	installationID, err := github.GetInstallationID(context.Background(), jwtToken, owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get installation ID: %w", err)
	}

	installationToken, err := github.GetInstallationToken(context.Background(), jwtToken, installationID)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}

	return installationToken, nil
}

// workflowTokenGitClient implements workflowtoken.GitClient
type workflowTokenGitClient struct{}

func (c *workflowTokenGitClient) ConfigureAuth(token string) error {
	cleanupStaleTokenRewrites()

	insteadOfKey := "url.https://x-access-token:" + token + "@github.com/.insteadOf"
	if err := git.Config(true, insteadOfKey, "https://github.com/"); err != nil {
		return fmt.Errorf("failed to configure git HTTPS authentication: %w", err)
	}

	if err := authenticateGHCLI(token); err != nil {
		return fmt.Errorf("failed to authenticate gh CLI: %w", err)
	}

	return nil
}

func cleanupStaleTokenRewrites() {
	out, err := git.ConfigList(true)
	if err != nil {
		return
	}

	for _, line := range strings.Split(out, "\n") {
		lower := strings.ToLower(line)
		if !strings.HasPrefix(lower, "url.https://x-access-token:") || !strings.Contains(lower, "github.com") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		git.ConfigUnset(true, key)
	}
}

func authenticateGHCLI(token string) error {
	loginCmd := exec.CommandContext(context.Background(), "gh", "auth", "login", "--with-token")
	loginCmd.Stdin = strings.NewReader(token)
	loginCmd.Stdout = os.Stdout
	loginCmd.Stderr = os.Stderr
	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("failed to authenticate gh CLI: %w", err)
	}
	return nil
}
