package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/orchestration/setup"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/workspace"
)

type SetupCmd struct {
	GithubKey   string `help:"Path to GitHub App private key (.pem file)" name:"github-key" optional:""`
	GithubToken string `help:"GitHub personal access token" name:"github-token" optional:""`
	Context     string `help:"The name of the Kubernetes context to use" name:"context" optional:""`
	Namespace   string `help:"The name of the Kubernetes namespace to use" short:"n" optional:""`
}

func (c *SetupCmd) Run() error {
	ctx := context.Background()

	out := output.NewClient(os.Stdout, os.Stderr, false)
	out.Info("Configuring credentials for Ralph remote execution...")

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}

	k8sClient := k8s.NewClient()

	cmd := &setup.SetupCmd{
		Readiness: &setupLocalReadinessClient{out: out},
		Ctx:       &setupContextClient{ctx: ctx, k8sClient: k8sClient, ralphConfig: ralphConfig},
		GitHub:    &setupGitHubClient{ctx: ctx, k8sClient: k8sClient, out: out},
		OpenCode:  &setupOpenCodeClient{ctx: ctx, k8sClient: k8sClient, out: out},
	}

	return cmd.Run(setup.Flags{
		Context:     c.Context,
		Namespace:   c.Namespace,
		GithubKey:   c.GithubKey,
		GithubToken: c.GithubToken,
	})
}

type setupLocalReadinessClient struct {
	out *output.Client
}

func (c *setupLocalReadinessClient) ConfirmGitReady() error {
	if !git.Installed() {
		return errors.New("git is not ready: git is not installed")
	}
	if !git.InsideRepository() {
		return errors.New("git is not ready: current directory is not inside a git repository")
	}
	if git.ConfigGet("user.name") == "" || git.ConfigGet("user.email") == "" {
		return errors.New("git is not ready: configure a git identity with user.name and user.email")
	}
	c.out.Success("git is ready")
	return nil
}

func (c *setupLocalReadinessClient) ConfirmGitHubCLIReady() error {
	if !github.GHInstalled() {
		return errors.New("GitHub CLI is not ready: gh is not installed")
	}
	if github.GHCliToken() == "" && os.Getenv("GITHUB_TOKEN") == "" {
		return errors.New("GitHub CLI is not ready: no token available - authenticate with gh auth login or set GITHUB_TOKEN")
	}
	c.out.Success("GitHub CLI is ready")
	return nil
}

func (c *setupLocalReadinessClient) ConfirmOpenCodeReady() error {
	if !opencode.Installed() {
		return errors.New("OpenCode is not ready: opencode is not installed")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("OpenCode is not ready: failed to get user home directory: %w", err)
	}
	authFilePath := filepath.Join(homeDir, ".local/share/opencode/auth.json")
	if _, err := workspace.ReadOpenCodeCredentials(authFilePath); err != nil {
		return fmt.Errorf("OpenCode is not ready: %w", err)
	}
	c.out.Success("OpenCode is ready")
	return nil
}

type setupContextClient struct {
	ctx         context.Context
	k8sClient   k8s.Client
	ralphConfig *config.RalphConfig
}

func (a *setupContextClient) Resolve(flagContext, flagNamespace string) (setup.K8sContext, error) {
	if flagNamespace == "" && (a.ralphConfig == nil || a.ralphConfig.Workflow.Namespace == "") {
		return setup.K8sContext{}, setup.ErrNoTargetedNamespace
	}
	k8sCtx, err := resolveKubeContext(a.ctx, a.k8sClient, a.ralphConfig, nil, flagContext, flagNamespace)
	if err != nil {
		return setup.K8sContext{}, err
	}
	return setup.K8sContext{Name: k8sCtx.Name, Namespace: k8sCtx.Namespace}, nil
}

type setupGitHubClient struct {
	ctx       context.Context
	k8sClient k8s.Client
	out       *output.Client
}

func (c *setupGitHubClient) SecretExists(k8sCtx setup.K8sContext) (bool, error) {
	return c.k8sClient.SecretExists(c.ctx, k8s.GitHubSecretName, k8sCtx.Namespace, k8sCtx.Name)
}

func (c *setupGitHubClient) Validate(keyPath string) error {
	c.out.Info("Validating credentials...")
	if err := github.ValidateAppCredentials(c.ctx, keyPath, config.DefaultAppID); err != nil {
		return err
	}
	c.out.Success("Credentials validated successfully")
	return nil
}

func (c *setupGitHubClient) Configure(k8sCtx setup.K8sContext, keyPath string) error {
	privateKeyBytes, err := github.ReadGitHubAppCredentials(keyPath)
	if err != nil {
		return err
	}

	c.out.Infof("Creating/updating Kubernetes secret '%s'...", k8s.GitHubSecretName)

	secretData := map[string]string{
		"app-id":      config.DefaultAppID,
		"private-key": string(privateKeyBytes),
	}

	if err := c.k8sClient.CreateOrUpdateSecret(c.ctx, k8s.GitHubSecretName, k8sCtx.Namespace, k8sCtx.Name, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	c.out.Successf("Secret '%s' created/updated successfully", k8s.GitHubSecretName)
	c.out.Infof("Configuration complete! The secret '%s' is ready for use in namespace '%s'.", k8s.GitHubSecretName, k8sCtx.Namespace)
	return nil
}

func (c *setupGitHubClient) TokenFromGHCli() string {
	return github.GHCliToken()
}

func (c *setupGitHubClient) TokenFromEnv() string {
	return os.Getenv("GITHUB_TOKEN")
}

func (c *setupGitHubClient) ConfigureToken(k8sCtx setup.K8sContext, token string) error {
	c.out.Infof("Creating/updating Kubernetes secret '%s'...", k8s.GitHubSecretName)

	secretData := map[string]string{
		"token": token,
	}

	if err := c.k8sClient.CreateOrUpdateSecret(c.ctx, k8s.GitHubSecretName, k8sCtx.Namespace, k8sCtx.Name, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	c.out.Successf("Secret '%s' created/updated successfully", k8s.GitHubSecretName)
	c.out.Infof("Configuration complete! The secret '%s' is ready for use in namespace '%s'.", k8s.GitHubSecretName, k8sCtx.Namespace)
	return nil
}

type setupOpenCodeClient struct {
	ctx       context.Context
	k8sClient k8s.Client
	out       *output.Client
}

func (c *setupOpenCodeClient) Configure(k8sCtx setup.K8sContext) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	authFilePath := homeDir + "/.local/share/opencode/auth.json"
	c.out.Infof("Reading OpenCode credentials from: %s", authFilePath)

	authFileContent, err := workspace.ReadOpenCodeCredentials(authFilePath)
	if err != nil {
		return err
	}

	c.out.Success("OpenCode credentials read successfully")

	c.out.Infof("Creating/updating Kubernetes secret '%s'...", k8s.OpenCodeSecretName)

	secretData := map[string]string{
		"auth.json": string(authFileContent),
	}

	if err := c.k8sClient.CreateOrUpdateSecret(c.ctx, k8s.OpenCodeSecretName, k8sCtx.Namespace, k8sCtx.Name, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	c.out.Successf("Secret '%s' created/updated successfully", k8s.OpenCodeSecretName)
	c.out.Infof("Configuration complete! The secret '%s' is ready for use in namespace '%s'.", k8s.OpenCodeSecretName, k8sCtx.Namespace)
	return nil
}
