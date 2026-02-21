package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/logger"
)

// ConfigGitCmd configures git credentials for Argo Workflows
type ConfigGitCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
}

// Run executes the config git command
func (c *ConfigGitCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Configuring Git credentials for Ralph remote execution...")
	fmt.Println()

	// Load context and namespace with priority: flags > .ralph/config.yaml > kubectl
	kubeContext, namespace, err := loadContextAndNamespace(ctx, c.Context, c.Namespace)
	if err != nil {
		return err
	}

	fmt.Println()

	// Get the current repository name from git remote
	repoName, _, err := k8s.GetGitHubRepo(ctx)
	if err != nil {
		logger.Warningf("Failed to detect GitHub repository: %v", err)
		repoName = "repo"
	}

	// Key title based on repository: ralph-{repo}
	keyTitle := fmt.Sprintf("ralph-%s", repoName)

	// Check if gh CLI is available
	ghAvailable := k8s.IsGHCLIAvailable(ctx)
	if ghAvailable {
		fmt.Println("GitHub CLI detected - will attempt automatic key management")
	} else {
		fmt.Println("GitHub CLI not found - will provide manual instructions")
	}
	fmt.Println()

	// If gh CLI is available, check for existing key and offer to delete it
	if ghAvailable {
		existingKeyID, err := k8s.FindGitHubSSHKey(ctx, keyTitle)
		if err != nil {
			logger.Warningf("Failed to check for existing SSH key: %v", err)
		} else if existingKeyID != "" {
			fmt.Printf("Found existing SSH key '%s' on GitHub\n", keyTitle)
			fmt.Print("Do you want to delete it and create a new one? (y/N): ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response == "y" || response == "yes" {
				fmt.Printf("Deleting existing SSH key '%s' from GitHub...\n", keyTitle)
				if err := k8s.DeleteGitHubSSHKey(ctx, existingKeyID); err != nil {
					logger.Warningf("Failed to delete existing key: %v (continuing anyway)", err)
				} else {
					fmt.Println("✓ Existing SSH key deleted")
				}
			}
			fmt.Println()
		}
	}

	fmt.Println("Generating SSH key pair...")

	// Generate SSH key pair
	privateKey, publicKey, err := k8s.GenerateSSHKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate SSH key pair: %w", err)
	}

	fmt.Println("✓ SSH key pair generated")
	fmt.Println()

	// Create or update the Kubernetes secret
	fmt.Printf("Creating/updating Kubernetes secret '%s'...\n", k8s.GitSecretName)

	secretData := map[string]string{
		"ssh-privatekey": privateKey,
	}

	if err := k8s.CreateOrUpdateSecret(ctx, k8s.GitSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	fmt.Printf("✓ Secret '%s' created/updated successfully\n", k8s.GitSecretName)
	fmt.Println()

	// If gh CLI is available, automatically add the key to GitHub
	if ghAvailable {
		fmt.Printf("Adding SSH key '%s' to GitHub...\n", keyTitle)
		if err := k8s.AddGitHubSSHKey(ctx, publicKey, keyTitle); err != nil {
			logger.Warningf("Failed to add SSH key to GitHub: %v", err)
			fmt.Println()
			fmt.Println("⚠ Automatic key addition failed. Please add manually:")
			fmt.Println()
			printManualSSHKeyInstructions(publicKey, keyTitle, namespace)
		} else {
			fmt.Printf("✓ SSH key '%s' added to GitHub successfully\n", keyTitle)
			fmt.Println()
			fmt.Printf("Configuration complete! The secret '%s' is ready for use in namespace '%s'.\n", k8s.GitSecretName, namespace)
		}
	} else {
		// No gh CLI - provide manual instructions
		printManualSSHKeyInstructions(publicKey, keyTitle, namespace)
	}

	return nil
}

// printManualSSHKeyInstructions prints instructions for manually adding SSH key
func printManualSSHKeyInstructions(publicKey, keyTitle, namespace string) {
	fmt.Println("Public SSH Key:")
	fmt.Println("===============")
	fmt.Println(publicKey)
	fmt.Println()

	fmt.Println("Next Steps:")
	fmt.Println("===========")
	fmt.Println("1. Copy the public key above")
	fmt.Println("2. Add it to your GitHub account SSH keys:")
	fmt.Println("   https://github.com/settings/ssh/new")
	fmt.Println()
	fmt.Printf("3. Use the title: %s\n", keyTitle)
	fmt.Println()
	fmt.Printf("Configuration complete! The secret '%s' is ready for use in namespace '%s'.\n", k8s.GitSecretName, namespace)
	fmt.Println()
	fmt.Println("Tip: Install GitHub CLI (gh) for automatic key management:")
	fmt.Println("  https://cli.github.com/")
}
