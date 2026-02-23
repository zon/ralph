package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/k8s"
)

// ConfigOpencodeCmd configures OpenCode credentials for Argo Workflows
type ConfigOpencodeCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
}

// Run executes the config opencode command
func (c *ConfigOpencodeCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Configuring OpenCode credentials for Ralph remote execution...")
	fmt.Println()

	// Load context and namespace with priority: flags > .ralph/config.yaml > kubectl
	kubeContext, namespace, err := loadContextAndNamespace(ctx, c.Context, c.Namespace)
	if err != nil {
		return err
	}

	fmt.Println()

	// Read OpenCode auth.json from user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	authFilePath := fmt.Sprintf("%s/.local/share/opencode/auth.json", homeDir)
	fmt.Printf("Reading OpenCode credentials from: %s\n", authFilePath)

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

	fmt.Println("✓ OpenCode credentials read successfully")
	fmt.Println()

	// Create or update the Kubernetes secret
	fmt.Printf("Creating/updating Kubernetes secret '%s'...\n", k8s.OpenCodeSecretName)

	secretData := map[string]string{
		"auth.json": string(authFileContent),
	}

	if err := k8s.CreateOrUpdateSecret(ctx, k8s.OpenCodeSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	fmt.Printf("✓ Secret '%s' created/updated successfully\n", k8s.OpenCodeSecretName)
	fmt.Println()

	fmt.Printf("Configuration complete! The secret '%s' is ready for use in namespace '%s'.\n", k8s.OpenCodeSecretName, namespace)

	return nil
}
