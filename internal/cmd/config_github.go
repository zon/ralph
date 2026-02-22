package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/logger"
	"golang.org/x/term"
)

// ConfigGithubCmd configures GitHub credentials for Argo Workflows
type ConfigGithubCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
}

// Run executes the config github command
func (c *ConfigGithubCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Configuring GitHub credentials for Ralph remote execution...")
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
		logger.Warningf("Failed to detect GitHub repository: %v", err)
		repoName = "repo"
	}

	// Token name format: ralph-{repo}
	tokenName := fmt.Sprintf("ralph-%s", repoName)

	// Output instructions for creating fine-grained token
	fmt.Println("GitHub Fine-Grained Personal Access Token Required")
	fmt.Println("===================================================")
	fmt.Println()
	fmt.Println("Ralph needs a GitHub personal access token to create pull requests.")
	fmt.Println()
	fmt.Println("Create a fine-grained personal access token:")
	fmt.Println()
	fmt.Println("1. Go to: https://github.com/settings/personal-access-tokens/new")
	fmt.Println()
	fmt.Printf("2. Token name: %s\n", tokenName)
	fmt.Println()
	fmt.Println("3. Expiration: Choose an appropriate expiration (90 days recommended)")
	fmt.Println()
	if repoOwner != "" && repoName != "repo" {
		fmt.Printf("4. Repository access: Only select repositories → %s/%s\n", repoOwner, repoName)
	} else {
		fmt.Println("4. Repository access: Only select repositories → Select your repository")
	}
	fmt.Println()
	fmt.Println("5. Permissions:")
	fmt.Println("   - Contents: Read and write")
	fmt.Println("   - Pull requests: Read and write")
	fmt.Println("   - Metadata: Read-only (automatically selected)")
	fmt.Println()
	fmt.Println("6. Click 'Generate token' and copy the token")
	fmt.Println()
	fmt.Println("Note: Fine-grained tokens are more secure than classic tokens as they")
	fmt.Println("      can be scoped to specific repositories with minimal permissions.")
	fmt.Println()

	// Prompt for GitHub token (hidden input)
	fmt.Print("Enter your GitHub personal access token: ")

	// Read token securely (hidden input)
	tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	fmt.Println() // Print newline after hidden input

	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	fmt.Println()

	// Create or update the Kubernetes secret
	fmt.Printf("Creating/updating Kubernetes secret '%s'...\n", k8s.GitHubSecretName)

	secretData := map[string]string{
		"token": token,
	}

	if err := k8s.CreateOrUpdateSecret(ctx, k8s.GitHubSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	fmt.Printf("✓ Secret '%s' created/updated successfully\n", k8s.GitHubSecretName)
	fmt.Println()

	fmt.Printf("Configuration complete! The secret '%s' is ready for use in namespace '%s'.\n", k8s.GitHubSecretName, namespace)
	fmt.Println()
	fmt.Printf("Remember: This token is named '%s' and should only have access to your repository.\n", tokenName)

	return nil
}
