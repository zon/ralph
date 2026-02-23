package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

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

	// Check if auth.json contains Anthropic OAuth credentials
	// If so, remove them to prevent OAuth refresh token conflicts between local and remote
	if err := removeAnthropicOAuthFromLocal(authFilePath, string(authFileContent)); err != nil {
		return fmt.Errorf("failed to handle local Anthropic OAuth credentials: %w", err)
	}

	fmt.Printf("Configuration complete! The secret '%s' is ready for use in namespace '%s'.\n", k8s.OpenCodeSecretName, namespace)

	return nil
}

// removeAnthropicOAuthFromLocal removes Anthropic OAuth credentials from local auth.json
// to prevent OAuth refresh token conflicts between local and remote execution
func removeAnthropicOAuthFromLocal(authFilePath, authContent string) error {
	// Parse the auth.json content
	var authData map[string]interface{}
	if err := json.Unmarshal([]byte(authContent), &authData); err != nil {
		return fmt.Errorf("failed to parse auth.json: %w", err)
	}

	// Check if Anthropic entry exists and is OAuth type
	anthropic, hasAnthropic := authData["anthropic"].(map[string]interface{})
	if !hasAnthropic {
		// No Anthropic entry, nothing to do
		return nil
	}

	authType, _ := anthropic["type"].(string)
	if authType != "oauth" {
		// Not OAuth, nothing to do (API keys don't have refresh token conflicts)
		return nil
	}

	// Remove the Anthropic entry
	delete(authData, "anthropic")

	// Write the modified auth.json back
	modifiedAuth, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal modified auth.json: %w", err)
	}

	if err := os.WriteFile(authFilePath, modifiedAuth, 0600); err != nil {
		return fmt.Errorf("failed to write modified auth.json: %w", err)
	}

	fmt.Println("⚠️  Removed Anthropic OAuth from local config to prevent token conflicts. Launching 'opencode auth login'...")

	// Launch opencode auth login to restore local access
	cmd := exec.Command("opencode", "auth", "login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: opencode auth login failed: %v\n", err)
		fmt.Println("You can run 'opencode auth login' manually later to restore local access.")
	}

	return nil
}
