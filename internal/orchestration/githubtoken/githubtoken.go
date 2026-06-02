package githubtoken

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type GitHubAuthClient interface {
	GetRepo(ctx context.Context) (owner, name string, err error)
	GenerateAppJWT(appID string, privateKeyPEM []byte) (string, error)
	GetInstallationID(ctx context.Context, jwtToken, owner, repo string) (int64, error)
	GetInstallationToken(ctx context.Context, jwtToken string, installationID int64) (string, error)
	ConfigureGitAuth(ctx context.Context, installationToken string) error
	AuthenticateGHCLI(ctx context.Context, installationToken string) error
}

type FsClient interface {
	ReadFile(path string) ([]byte, error)
}

type Cmd struct {
	ghClient GitHubAuthClient
	fsClient FsClient
}

func New(ghClient GitHubAuthClient, fsClient FsClient) *Cmd {
	return &Cmd{
		ghClient: ghClient,
		fsClient: fsClient,
	}
}

func (c *Cmd) Run(ctx context.Context, owner, repo, secretsDir string) error {
	resolvedOwner, resolvedRepo, err := c.resolveRepoDetails(ctx, owner, repo)
	if err != nil {
		return err
	}

	appID, privateKeyBytes, err := c.readAppCredentials(secretsDir)
	if err != nil {
		return err
	}

	installationToken, err := c.obtainInstallationToken(ctx, resolvedOwner, resolvedRepo, appID, privateKeyBytes)
	if err != nil {
		return err
	}

	if err := c.ghClient.ConfigureGitAuth(ctx, installationToken); err != nil {
		return fmt.Errorf("failed to configure git HTTPS authentication: %w", err)
	}

	if err := c.ghClient.AuthenticateGHCLI(ctx, installationToken); err != nil {
		return fmt.Errorf("failed to authenticate gh CLI: %w", err)
	}

	return nil
}

func (c *Cmd) resolveRepoDetails(ctx context.Context, owner, repo string) (string, string, error) {
	if owner == "" || repo == "" {
		detectedOwner, detectedRepo, err := c.ghClient.GetRepo(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to autodetect repository from git remote: %w", err)
		}
		if owner == "" {
			owner = detectedOwner
		}
		if repo == "" {
			repo = detectedRepo
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

func (c *Cmd) readAppCredentials(secretsDir string) (string, []byte, error) {
	appIDPath := filepath.Join(secretsDir, "app-id")
	appIDBytes, err := c.fsClient.ReadFile(appIDPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read app ID from %s: %w", appIDPath, err)
	}
	appID := strings.TrimSpace(string(appIDBytes))
	if appID == "" {
		return "", nil, fmt.Errorf("app ID is empty in %s", appIDPath)
	}

	privateKeyPath := filepath.Join(secretsDir, "private-key")
	privateKeyBytes, err := c.fsClient.ReadFile(privateKeyPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read private key from %s: %w", privateKeyPath, err)
	}
	if len(privateKeyBytes) == 0 {
		return "", nil, fmt.Errorf("private key is empty in %s", privateKeyPath)
	}

	return appID, privateKeyBytes, nil
}

func (c *Cmd) obtainInstallationToken(ctx context.Context, owner, repo, appID string, privateKeyBytes []byte) (string, error) {
	jwtToken, err := c.ghClient.GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	installationID, err := c.ghClient.GetInstallationID(ctx, jwtToken, owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get installation ID: %w", err)
	}

	installationToken, err := c.ghClient.GetInstallationToken(ctx, jwtToken, installationID)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}

	return installationToken, nil
}
