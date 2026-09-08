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

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}

	k8sClient := k8s.NewClient()

	// A namespace is targeted by the --namespace flag or workflow.namespace in
	// .ralph/config.yaml. setupContextClient.Resolve skips preparation under
	// the same condition, so only a non-empty target reports secret results.
	targetNamespace := c.Namespace
	if targetNamespace == "" && ralphConfig != nil {
		targetNamespace = ralphConfig.Workflow.Namespace
	}

	cmd := &setup.SetupCmd{
		Readiness: &setupLocalReadinessClient{out: out},
		Ctx:       &setupContextClient{ctx: ctx, k8sClient: k8sClient, ralphConfig: ralphConfig},
		GitHub:    &setupGitHubClient{ctx: ctx, k8sClient: k8sClient, out: out},
		OpenCode:  &setupOpenCodeClient{ctx: ctx, k8sClient: k8sClient, out: out},
	}

	if err := cmd.Run(setup.Flags{
		Context:     c.Context,
		Namespace:   c.Namespace,
		GithubKey:   c.GithubKey,
		GithubToken: c.GithubToken,
	}); err != nil {
		return err
	}

	if targetNamespace != "" {
		out.Successf("%s/%s secret ready", targetNamespace, k8s.GitHubSecretName)
		out.Successf("%s/%s secret ready", targetNamespace, k8s.OpenCodeSecretName)
	}
	return nil
}

type setupLocalReadinessClient struct {
	out *output.Client
}

// notReady prints the failed check line for cmd and returns reason as the
// error the caller reports as the brief description.
func (c *setupLocalReadinessClient) notReady(cmd, reason string) error {
	c.out.Error("\u2717 " + cmd + " not ready")
	return errors.New(reason)
}

func (c *setupLocalReadinessClient) ConfirmGitReady() error {
	if !git.Installed() {
		return c.notReady("git", "git is not installed")
	}
	if !git.InsideRepository() {
		return c.notReady("git", "not inside a git repository")
	}
	if git.ConfigGet("user.name") == "" || git.ConfigGet("user.email") == "" {
		return c.notReady("git", "configure a git identity with user.name and user.email")
	}
	c.out.Success("git ready")
	return nil
}

func (c *setupLocalReadinessClient) ConfirmGitHubCLIReady() error {
	if !github.GHInstalled() {
		return c.notReady("gh", "gh is not installed")
	}
	if github.GHCliToken() == "" && os.Getenv("GITHUB_TOKEN") == "" {
		return c.notReady("gh", "no token available - authenticate with gh auth login or set GITHUB_TOKEN")
	}
	c.out.Success("gh ready")
	return nil
}

func (c *setupLocalReadinessClient) ConfirmOpenCodeReady() error {
	if !opencode.Installed() {
		return c.notReady("opencode", "opencode is not installed")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return c.notReady("opencode", fmt.Sprintf("failed to get user home directory: %v", err))
	}
	authFilePath := filepath.Join(homeDir, ".local/share/opencode/auth.json")
	if _, err := workspace.ReadOpenCodeCredentials(authFilePath); err != nil {
		return c.notReady("opencode", err.Error())
	}
	c.out.Success("opencode ready")
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
	return github.ValidateAppCredentials(c.ctx, keyPath, config.DefaultAppID)
}

// reportSecretNotReady prints the failed check line for the GitHub credentials
// secret and returns err as the description the caller reports.
func (c *setupGitHubClient) reportSecretNotReady(k8sCtx setup.K8sContext, err error) error {
	c.out.Errorf("\u2717 %s/%s secret not ready", k8sCtx.Namespace, k8s.GitHubSecretName)
	return err
}

func (c *setupGitHubClient) Configure(k8sCtx setup.K8sContext, keyPath string) error {
	privateKeyBytes, err := github.ReadGitHubAppCredentials(keyPath)
	if err != nil {
		return c.reportSecretNotReady(k8sCtx, err)
	}

	secretData := map[string]string{
		"app-id":      config.DefaultAppID,
		"private-key": string(privateKeyBytes),
	}

	if err := c.k8sClient.CreateOrUpdateSecret(c.ctx, k8s.GitHubSecretName, k8sCtx.Namespace, k8sCtx.Name, secretData); err != nil {
		return c.reportSecretNotReady(k8sCtx, fmt.Errorf("failed to create/update secret: %w", err))
	}

	return nil
}

func (c *setupGitHubClient) TokenFromGHCli() string {
	return github.GHCliToken()
}

func (c *setupGitHubClient) TokenFromEnv() string {
	return os.Getenv("GITHUB_TOKEN")
}

func (c *setupGitHubClient) ConfigureToken(k8sCtx setup.K8sContext, token string) error {
	secretData := map[string]string{
		"token": token,
	}

	if err := c.k8sClient.CreateOrUpdateSecret(c.ctx, k8s.GitHubSecretName, k8sCtx.Namespace, k8sCtx.Name, secretData); err != nil {
		return c.reportSecretNotReady(k8sCtx, fmt.Errorf("failed to create/update secret: %w", err))
	}

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
		return c.reportSecretNotReady(k8sCtx, fmt.Errorf("failed to get user home directory: %w", err))
	}

	authFilePath := filepath.Join(homeDir, ".local/share/opencode/auth.json")

	authFileContent, err := workspace.ReadOpenCodeCredentials(authFilePath)
	if err != nil {
		return c.reportSecretNotReady(k8sCtx, err)
	}

	secretData := map[string]string{
		"auth.json": string(authFileContent),
	}

	if err := c.k8sClient.CreateOrUpdateSecret(c.ctx, k8s.OpenCodeSecretName, k8sCtx.Namespace, k8sCtx.Name, secretData); err != nil {
		return c.reportSecretNotReady(k8sCtx, fmt.Errorf("failed to create/update secret: %w", err))
	}

	return nil
}

// reportSecretNotReady prints the failed check line for the OpenCode
// credentials secret and returns err as the description the caller reports.
func (c *setupOpenCodeClient) reportSecretNotReady(k8sCtx setup.K8sContext, err error) error {
	c.out.Errorf("\u2717 %s/%s secret not ready", k8sCtx.Namespace, k8s.OpenCodeSecretName)
	return err
}
