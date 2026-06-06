package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/output"
	setconfig "github.com/zon/ralph/internal/orchestration/setconfig"
)

type SetConfigCmd struct {
	GithubKey string `help:"Path to GitHub App private key (.pem file)" name:"github-key" optional:""`
	Context   string `help:"Kubernetes context to use" name:"context" optional:""`
	Namespace string `help:"Kubernetes namespace to use" short:"n" optional:""`
}

func (c *SetConfigCmd) Run() error {
	ctx := context.Background()

	out := output.NewClient(os.Stdout, os.Stderr, false)
	out.Info("Configuring credentials for Ralph remote execution...")

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}

	k8sClient := k8s.NewClient()

	cmd := &setconfig.SetConfigCmd{
		Ctx:      &setconfigContextClient{ctx: ctx, k8sClient: k8sClient, ralphConfig: ralphConfig},
		GitHub:   &setconfigGitHubClient{ctx: ctx, k8sClient: k8sClient, out: out},
		OpenCode: &setconfigOpenCodeClient{ctx: ctx, k8sClient: k8sClient, out: out},
	}

	return cmd.Run(setconfig.Flags{
		Context:   c.Context,
		Namespace: c.Namespace,
		GithubKey: c.GithubKey,
	})
}

type setconfigContextClient struct {
	ctx         context.Context
	k8sClient   k8s.Client
	ralphConfig *config.RalphConfig
}

func (a *setconfigContextClient) Resolve(flagContext, flagNamespace string) (setconfig.K8sContext, error) {
	k8sCtx, err := resolveKubeContext(a.ctx, a.k8sClient, a.ralphConfig, nil, flagContext, flagNamespace)
	if err != nil {
		return setconfig.K8sContext{}, err
	}
	return setconfig.K8sContext{Name: k8sCtx.Name, Namespace: k8sCtx.Namespace}, nil
}

type setconfigGitHubClient struct {
	ctx       context.Context
	k8sClient k8s.Client
	out       *output.Client
}

func (c *setconfigGitHubClient) SecretExists(k8sCtx setconfig.K8sContext) (bool, error) {
	args := []string{"get", "secret", k8s.GitHubSecretName, "-n", k8sCtx.Namespace}
	if k8sCtx.Name != "" {
		args = append(args, "--context", k8sCtx.Name)
	}
	cmd := exec.CommandContext(c.ctx, "kubectl", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if bytes.Contains(stderr.Bytes(), []byte("not found")) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check secret: %w (stderr: %s)", err, stderr.String())
	}
	return true, nil
}

func (c *setconfigGitHubClient) Validate(keyPath string) error {
	privateKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}
	if len(privateKeyBytes) == 0 {
		return fmt.Errorf("private key file is empty")
	}

	c.out.Info("Validating credentials...")

	repo, err := github.GetRepo(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to detect GitHub repository: %w", err)
	}

	appID := config.DefaultAppID
	jwtToken, err := github.GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to generate JWT for validation: %w", err)
	}

	installationID, err := github.GetInstallationID(c.ctx, jwtToken, repo.Owner, repo.Name)
	if err != nil {
		return fmt.Errorf("failed to get installation ID: %w", err)
	}

	_, err = github.GetInstallationToken(c.ctx, jwtToken, installationID)
	if err != nil {
		return fmt.Errorf("failed to get installation token: %w", err)
	}

	c.out.Success("Credentials validated successfully")
	return nil
}

func (c *setconfigGitHubClient) Configure(k8sCtx setconfig.K8sContext, keyPath string) error {
	privateKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}

	c.out.Infof("Creating/updating Kubernetes secret '%s'...", k8s.GitHubSecretName)

	appID := config.DefaultAppID
	secretData := map[string]string{
		"app-id":      appID,
		"private-key": string(privateKeyBytes),
	}

	if err := c.k8sClient.CreateOrUpdateSecret(c.ctx, k8s.GitHubSecretName, k8sCtx.Namespace, k8sCtx.Name, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	c.out.Successf("Secret '%s' created/updated successfully", k8s.GitHubSecretName)
	c.out.Infof("Configuration complete! The secret '%s' is ready for use in namespace '%s'.", k8s.GitHubSecretName, k8sCtx.Namespace)
	return nil
}

type setconfigOpenCodeClient struct {
	ctx       context.Context
	k8sClient k8s.Client
	out       *output.Client
}

func (c *setconfigOpenCodeClient) Configure(k8sCtx setconfig.K8sContext) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	authFilePath := fmt.Sprintf("%s/.local/share/opencode/auth.json", homeDir)
	c.out.Infof("Reading OpenCode credentials from: %s", authFilePath)

	authFileContent, err := os.ReadFile(authFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("OpenCode auth.json not found at %s\n\nPlease ensure OpenCode is configured and the auth.json file exists.", authFilePath)
		}
		return fmt.Errorf("failed to read auth.json: %w", err)
	}

	if len(authFileContent) == 0 {
		return fmt.Errorf("auth.json is empty at %s", authFilePath)
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
