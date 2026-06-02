package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/git"
	internalgithub "github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/orchestration/githubtoken"
)

func (g *GithubTokenCmd) Run() error {
	ctx := context.Background()

	orchestrator := newGithubTokenOrchestrator()
	return orchestrator.Run(ctx, g.Owner, g.Repo, g.SecretsDir)
}

type githubTokenGitHubClientAdapter struct{}

func (a *githubTokenGitHubClientAdapter) GetRepo(ctx context.Context) (string, string, error) {
	repo, err := internalgithub.GetRepo(ctx)
	if err != nil {
		return "", "", err
	}
	return repo.Owner, repo.Name, nil
}

func (a *githubTokenGitHubClientAdapter) GenerateAppJWT(appID string, privateKeyPEM []byte) (string, error) {
	return internalgithub.GenerateAppJWT(appID, privateKeyPEM)
}

func (a *githubTokenGitHubClientAdapter) GetInstallationID(ctx context.Context, jwtToken, owner, repo string) (int64, error) {
	return internalgithub.GetInstallationID(ctx, jwtToken, owner, repo)
}

func (a *githubTokenGitHubClientAdapter) GetInstallationToken(ctx context.Context, jwtToken string, installationID int64) (string, error) {
	return internalgithub.GetInstallationToken(ctx, jwtToken, installationID)
}

func (a *githubTokenGitHubClientAdapter) ConfigureGitAuth(ctx context.Context, installationToken string) error {
	cleanupStaleTokenRewrites(ctx)

	insteadOfKey := "url.https://x-access-token:" + installationToken + "@github.com/.insteadOf"
	if err := git.Config(true, insteadOfKey, "https://github.com/"); err != nil {
		return fmt.Errorf("failed to configure git HTTPS authentication: %w", err)
	}
	return nil
}

func (a *githubTokenGitHubClientAdapter) AuthenticateGHCLI(ctx context.Context, installationToken string) error {
	loginCmd := exec.CommandContext(ctx, "gh", "auth", "login", "--with-token")
	loginCmd.Stdin = strings.NewReader(installationToken)
	loginCmd.Stdout = os.Stdout
	loginCmd.Stderr = os.Stderr
	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("failed to authenticate gh CLI: %w", err)
	}
	return nil
}

func cleanupStaleTokenRewrites(ctx context.Context) {
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

type githubTokenFsClientAdapter struct{}

func (a *githubTokenFsClientAdapter) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func newGithubTokenOrchestrator() *githubtoken.Cmd {
	return githubtoken.New(
		&githubTokenGitHubClientAdapter{},
		&githubTokenFsClientAdapter{},
	)
}
