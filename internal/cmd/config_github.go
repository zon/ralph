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

	fmt.Println("Configuring GitHub App credentials for Ralph remote execution...")
	fmt.Println()

	// Load context and namespace with priority: flags > .ralph/config.yaml > kubectl
	kubeContext, namespace, err := loadContextAndNamespace(ctx, c.Context, c.Namespace)
	if err != nil {
		return err
	}

	fmt.Println()

	// Get the current repository name from git remote
	repoName, repoOwner, err := github.GetRepo(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect GitHub repository: %w", err)
	}

	_ = repoName
	_ = repoOwner

	appID := config.DefaultAppID

	// Read private key
	privateKeyBytes, err := os.ReadFile(c.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}
	if len(privateKeyBytes) == 0 {
		return fmt.Errorf("private key file is empty")
	}

	// Validate credentials by generating a test installation token
	fmt.Println("Validating credentials...")
	jwtToken, err := github.GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to generate JWT for validation: %w", err)
	}

	// Get installation ID
	installationID, err := github.GetInstallationID(ctx, jwtToken, repoOwner, repoName)
	if err != nil {
		return fmt.Errorf("failed to get installation ID: %w", err)
	}

	// Get installation token (validate it works)
	_, err = github.GetInstallationToken(ctx, jwtToken, installationID)
	if err != nil {
		return fmt.Errorf("failed to get installation token: %w", err)
	}

	fmt.Println("✓ Credentials validated successfully")
	fmt.Println()

	// Create or update the Kubernetes secret
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
	fmt.Println("Note: GitHub App credentials are not tied to any user account.")

	return nil
}
