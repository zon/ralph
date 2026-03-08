package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
)

// ConfigGithubCmd configures GitHub credentials for Argo Workflows
type ConfigGithubCmd struct {
	PrivateKey string `arg:"" help:"Path to GitHub App private key (.pem file)" type:"existingfile"`
	Context    string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace  string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
}

// Run executes the config github command
func (c *ConfigGithubCmd) Run() error {
	ctx := context.Background()

	c.printHeader()

	kubeContext, namespace, err := loadContextAndNamespace(ctx, c.Context, c.Namespace)
	if err != nil {
		return err
	}

	fmt.Println()

	repoName, repoOwner, err := c.detectRepo(ctx)
	if err != nil {
		return err
	}

	_ = repoName
	_ = repoOwner

	privateKeyBytes, err := c.readAndValidatePrivateKey()
	if err != nil {
		return err
	}

	if err := c.validateCredentials(ctx, repoOwner, repoName, privateKeyBytes); err != nil {
		return err
	}

	appID := config.DefaultAppID

	if err := c.createK8sSecret(ctx, kubeContext, namespace, appID, privateKeyBytes); err != nil {
		return err
	}

	fmt.Println("Note: GitHub App credentials are not tied to any user account.")

	return nil
}

func (c *ConfigGithubCmd) printHeader() {
	fmt.Println("Configuring GitHub App credentials for Ralph remote execution...")
	fmt.Println()
}

func (c *ConfigGithubCmd) detectRepo(ctx context.Context) (string, string, error) {
	repoName, repoOwner, err := github.GetRepo(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to detect GitHub repository: %w", err)
	}
	return repoName, repoOwner, nil
}

func (c *ConfigGithubCmd) readAndValidatePrivateKey() ([]byte, error) {
	privateKeyBytes, err := os.ReadFile(c.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}
	if len(privateKeyBytes) == 0 {
		return nil, fmt.Errorf("private key file is empty")
	}
	return privateKeyBytes, nil
}

func (c *ConfigGithubCmd) validateCredentials(ctx context.Context, repoOwner, repoName string, privateKeyBytes []byte) error {
	fmt.Println("Validating credentials...")
	appID := config.DefaultAppID

	jwtToken, err := github.GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to generate JWT for validation: %w", err)
	}

	installationID, err := github.GetInstallationID(ctx, jwtToken, repoOwner, repoName)
	if err != nil {
		return fmt.Errorf("failed to get installation ID: %w", err)
	}

	_, err = github.GetInstallationToken(ctx, jwtToken, installationID)
	if err != nil {
		return fmt.Errorf("failed to get installation token: %w", err)
	}

	fmt.Println("✓ Credentials validated successfully")
	fmt.Println()
	return nil
}

func (c *ConfigGithubCmd) createK8sSecret(ctx context.Context, kubeContext, namespace, appID string, privateKeyBytes []byte) error {
	fmt.Printf("Creating/updating Kubernetes secret '%s'...\n", k8s.GitHubSecretName)

	secretData := map[string]string{
		"app-id":      appID,
		"private-key": string(privateKeyBytes),
	}

	if err := k8s.CreateOrUpdateSecret(ctx, k8s.GitHubSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	fmt.Printf("✓ Secret '%s' created/updated successfully\n", k8s.GitHubSecretName)
	fmt.Println()

	fmt.Printf("Configuration complete! The secret '%s' is ready for use in namespace '%s'.\n", k8s.GitHubSecretName, namespace)
	fmt.Println()
	return nil
}
