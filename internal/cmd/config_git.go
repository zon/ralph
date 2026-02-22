package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
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

	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	githubUser := ralphConfig.Workflow.GitUser.Name

	logger.Verbosef("GitHub user: %s", githubUser)

	// Get the current repository name from git remote
	repoName, _, err := github.GetRepo(ctx)
	if err != nil {
		logger.Warningf("Failed to detect GitHub repository: %v", err)
		repoName = "repo"
	}

	// Key title based on repository: ralph-{repo}
	keyTitle := fmt.Sprintf("ralph-%s", repoName)

	// Check if gh CLI is available
	ghAvailable := github.IsGHCLIAvailable(ctx)
	if ghAvailable {
		fmt.Println("GitHub CLI detected - will attempt automatic key management")
	} else {
		fmt.Println("GitHub CLI not found - will provide manual instructions")
	}
	fmt.Println()

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

	// If gh CLI is available, manage SSH key on GitHub
	if ghAvailable {
		if err := manageGitHubSSHKey(ctx, publicKey, keyTitle, githubUser, namespace); err != nil {
			return fmt.Errorf("failed to manage SSH key on GitHub: %w", err)
		}
	} else {
		// No gh CLI - provide manual instructions
		printManualSSHKeyInstructions(publicKey, keyTitle, namespace)
	}

	return nil
}

// manageGitHubSSHKey switches to targetUser, optionally deletes an existing key,
// adds the new public key, then restores the original authenticated user.
func manageGitHubSSHKey(ctx context.Context, publicKey, keyTitle, targetUser, namespace string) error {
	originalUser, err := github.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine current GitHub user: %w", err)
	}

	if !strings.EqualFold(originalUser, targetUser) {
		if err := github.SwitchUser(ctx, targetUser); err != nil {
			return err
		}
		defer func() {
			if switchErr := github.SwitchUser(ctx, originalUser); switchErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to restore GitHub user to %q: %v\n", originalUser, switchErr)
			}
		}()
	}

	// Check for existing key and offer to delete it
	existingKeyID, err := github.FindSSHKey(ctx, keyTitle)
	if err != nil {
		return fmt.Errorf("failed to check for existing SSH key: %w", err)
	} else if existingKeyID != "" {
		fmt.Printf("Found existing SSH key '%s' on GitHub\n", keyTitle)
		fmt.Print("Do you want to delete it and create a new one? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response == "y" || response == "yes" {
			fmt.Printf("Deleting existing SSH key '%s' from GitHub...\n", keyTitle)
			if err := github.DeleteSSHKey(ctx, existingKeyID); err != nil {
				logger.Warningf("Failed to delete existing key: %v (continuing anyway)", err)
			} else {
				fmt.Println("✓ Existing SSH key deleted")
			}
		}
		fmt.Println()
	}

	fmt.Printf("Adding SSH key '%s' to GitHub user '%s'...\n", keyTitle, targetUser)
	cmd := exec.CommandContext(ctx, "gh", "ssh-key", "add", "-", "-t", keyTitle)
	cmd.Stdin = strings.NewReader(publicKey)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add SSH key: %w (stderr: %s)", err, stderr.String())
	}

	fmt.Printf("✓ SSH key '%s' added to GitHub successfully\n", keyTitle)
	fmt.Println()
	fmt.Printf("Configuration complete! The secret '%s' is ready for use in namespace '%s'.\n", k8s.GitSecretName, namespace)
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
