package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/orchestration/setremote"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/workspace"
)

type SetRemoteCmd struct {
	GithubKey   string `help:"Path to GitHub App private key (.pem file)" name:"github-key" optional:""`
	GithubToken string `help:"GitHub personal access token" name:"github-token" optional:""`
	Context     string `help:"The name of the Kubernetes context to use" name:"context" optional:""`
	Namespace   string `help:"The name of the Kubernetes namespace to use" short:"n" optional:""`
}

func (c *SetRemoteCmd) Run() error {
	ctx := context.Background()

	out := output.NewClient(os.Stdout, os.Stderr, false)
	out.Info("Configuring credentials for Ralph remote execution...")

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}

	k8sClient := k8s.NewClient()

	cmd := &setremote.SetRemoteCmd{
		Ctx:      &setremoteContextClient{ctx: ctx, k8sClient: k8sClient, ralphConfig: ralphConfig},
		GitHub:   &setremoteGitHubClient{ctx: ctx, k8sClient: k8sClient, out: out},
		OpenCode: &setremoteOpenCodeClient{ctx: ctx, k8sClient: k8sClient, out: out},
	}

	return cmd.Run(setremote.Flags{
		Context:     c.Context,
		Namespace:   c.Namespace,
		GithubKey:   c.GithubKey,
		GithubToken: c.GithubToken,
	})
}

type setremoteContextClient struct {
	ctx         context.Context
	k8sClient   k8s.Client
	ralphConfig *config.RalphConfig
}

func (a *setremoteContextClient) Resolve(flagContext, flagNamespace string) (setremote.K8sContext, error) {
	k8sCtx, err := resolveKubeContext(a.ctx, a.k8sClient, a.ralphConfig, nil, flagContext, flagNamespace)
	if err != nil {
		return setremote.K8sContext{}, err
	}
	return setremote.K8sContext{Name: k8sCtx.Name, Namespace: k8sCtx.Namespace}, nil
}

type setremoteGitHubClient struct {
	ctx       context.Context
	k8sClient k8s.Client
	out       *output.Client
}

func (c *setremoteGitHubClient) SecretExists(k8sCtx setremote.K8sContext) (bool, error) {
	return c.k8sClient.SecretExists(c.ctx, k8s.GitHubSecretName, k8sCtx.Namespace, k8sCtx.Name)
}

func (c *setremoteGitHubClient) Validate(keyPath string) error {
	c.out.Info("Validating credentials...")
	if err := github.ValidateAppCredentials(c.ctx, keyPath, config.DefaultAppID); err != nil {
		return err
	}
	c.out.Success("Credentials validated successfully")
	return nil
}

func (c *setremoteGitHubClient) Configure(k8sCtx setremote.K8sContext, keyPath string) error {
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

func (c *setremoteGitHubClient) TokenFromGHCli() string {
	return github.GHCliToken()
}

func (c *setremoteGitHubClient) TokenFromEnv() string {
	return os.Getenv("GITHUB_TOKEN")
}

func (c *setremoteGitHubClient) ConfigureToken(k8sCtx setremote.K8sContext, token string) error {
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

type setremoteOpenCodeClient struct {
	ctx       context.Context
	k8sClient k8s.Client
	out       *output.Client
}

func (c *setremoteOpenCodeClient) Configure(k8sCtx setremote.K8sContext) error {
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
