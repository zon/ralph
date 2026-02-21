package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/requirement"
	"golang.org/x/term"
)

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	// Subcommands
	Run    RunCmd    `cmd:"" default:"withargs" help:"Execute ralph with a project file (default command)"`
	Config ConfigCmd `cmd:"" help:"Configure credentials for remote execution"`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// RunCmd is the default command for executing ralph
type RunCmd struct {
	WorkingDir    string `help:"Working directory to run ralph in" type:"path" short:"C"`
	ProjectFile   string `arg:"" optional:"" help:"Path to project YAML file" type:"path"`
	Once          bool   `help:"Single development iteration mode" default:"false"`
	MaxIterations int    `help:"Maximum number of development iterations (not applicable with --once)" default:"10"`
	DryRun        bool   `help:"Simulate execution without making changes" default:"false"`
	NoNotify      bool   `help:"Disable desktop notifications" default:"false"`
	NoServices    bool   `help:"Skip service startup" default:"false"`
	Verbose       bool   `help:"Enable verbose logging" default:"false"`
	Remote        bool   `help:"Execute workflow in Argo Workflows (incompatible with --once)" default:"false"`
	Watch         bool   `help:"Watch workflow execution (only applicable with --remote)" default:"false"`
	ShowVersion   bool   `help:"Show version information" short:"v" name:"version"`
	Instructions  string `help:"Path to an instructions file that overrides the default agent instructions" type:"path" optional:""`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// ConfigCmd defines the config subcommand group
type ConfigCmd struct {
	Git      ConfigGitCmd      `cmd:"" help:"Configure git credentials for remote execution"`
	Github   ConfigGithubCmd   `cmd:"" help:"Configure GitHub credentials for remote execution"`
	Opencode ConfigOpencodeCmd `cmd:"" help:"Configure OpenCode credentials for remote execution"`
}

// loadContextAndNamespace loads the Kubernetes context and namespace with the following priority:
// 1. Command-line flags (if provided)
// 2. .ralph/config.yaml (workflow.context and workflow.namespace)
// 3. kubectl configuration (current context and context namespace)
// 4. Default namespace ("default")
// Returns: kubeContext, namespace, error
func loadContextAndNamespace(ctx context.Context, flagContext, flagNamespace string) (string, string, error) {
	// Try to load .ralph/config.yaml for defaults
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		logger.Verbosef("Failed to load .ralph/config.yaml: %v (using kubectl config)", err)
	}

	// Determine the Kubernetes context
	var kubeContext string
	var contextSource string

	if flagContext != "" {
		// Command-line flag takes highest priority
		kubeContext = flagContext
		contextSource = "flag"
		logger.Verbosef("Using Kubernetes context: %s", kubeContext)
	} else if ralphConfig != nil && ralphConfig.Workflow.Context != "" {
		// .ralph/config.yaml is second priority
		kubeContext = ralphConfig.Workflow.Context
		contextSource = ".ralph/config.yaml"
		logger.Verbosef("Using context from .ralph/config.yaml: %s", kubeContext)
	} else {
		// Fall back to kubectl current context
		currentCtx, err := k8s.GetCurrentContext(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured.", err)
		}
		kubeContext = currentCtx
		contextSource = "kubectl"
		logger.Verbosef("Using current Kubernetes context: %s", kubeContext)
	}

	// Determine the namespace
	var namespace string

	if flagNamespace != "" {
		// Command-line flag takes highest priority
		namespace = flagNamespace
		logger.Verbosef("Using namespace: %s", namespace)
	} else if ralphConfig != nil && ralphConfig.Workflow.Namespace != "" {
		// .ralph/config.yaml is second priority
		namespace = ralphConfig.Workflow.Namespace
		if contextSource == ".ralph/config.yaml" {
			logger.Verbosef("Using namespace from .ralph/config.yaml: %s", namespace)
		} else {
			logger.Verbosef("Using namespace from .ralph/config.yaml: %s (context from %s)", namespace, contextSource)
		}
	} else {
		// Fall back to kubectl context namespace
		ns, err := k8s.GetNamespaceForContext(ctx, kubeContext)
		if err != nil {
			logger.Verbosef("Failed to get namespace for context: %v", err)
		}
		if ns == "" {
			namespace = "default"
			logger.Verbosef("Using namespace: %s (default)", namespace)
		} else {
			namespace = ns
			logger.Verbosef("Using namespace: %s (from kubectl context)", namespace)
		}
	}

	return kubeContext, namespace, nil
}

// ConfigGitCmd configures git credentials for Argo Workflows
type ConfigGitCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
}

// ConfigGithubCmd configures GitHub credentials for Argo Workflows
type ConfigGithubCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
}

// ConfigOpencodeCmd configures OpenCode credentials for Argo Workflows
type ConfigOpencodeCmd struct {
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
	repoName, repoOwner, err := k8s.GetGitHubRepo(ctx)
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

// SetVersion sets the version information
func (c *Cmd) SetVersion(version, date string) {
	c.version = version
	c.date = date
	c.Run.version = version
	c.Run.date = date
}

// SetCleanupRegistrar sets the cleanup registrar function
func (c *Cmd) SetCleanupRegistrar(cleanupRegistrar func(func())) {
	c.cleanupRegistrar = cleanupRegistrar
	c.Run.cleanupRegistrar = cleanupRegistrar
}

// Run executes the run command (implements kong.Run interface)
func (r *RunCmd) Run() error {
	// Handle version flag
	if r.ShowVersion {
		if r.date != "unknown" {
			fmt.Printf("ralph version %s (%s)\n", r.version, r.date)
		} else {
			fmt.Printf("ralph version %s\n", r.version)
		}
		return nil
	}

	// Change working directory if specified
	if r.WorkingDir != "" {
		if err := os.Chdir(r.WorkingDir); err != nil {
			return fmt.Errorf("failed to change to working directory %s: %w", r.WorkingDir, err)
		}
	}

	// Validate required fields
	if r.ProjectFile == "" {
		return fmt.Errorf("project file required (see --help)")
	}

	// Validate flag combinations
	if r.Watch && !r.Remote {
		return fmt.Errorf("--watch flag is only applicable with --remote flag")
	}

	if r.Remote && r.Once {
		return fmt.Errorf("--remote flag is incompatible with --once flag")
	}

	// Create execution context
	ctx := &execcontext.Context{
		ProjectFile:   r.ProjectFile,
		MaxIterations: r.MaxIterations,
		DryRun:        r.DryRun,
		Verbose:       r.Verbose,
		NoNotify:      r.NoNotify,
		NoServices:    r.NoServices,
		Remote:        r.Remote,
		Watch:         r.Watch,
		Instructions:  r.Instructions,
	}

	if r.Once {
		// Execute single iteration mode
		// Load project for notification
		project, err := config.LoadProject(ctx.ProjectFile)
		if err != nil {
			return fmt.Errorf("failed to load project: %w", err)
		}

		if err := requirement.Execute(ctx, r.cleanupRegistrar); err != nil {
			notify.Error(project.Name, ctx.ShouldNotify() && !ctx.IsDryRun())
			return err
		}

		notify.Success(project.Name, ctx.ShouldNotify() && !ctx.IsDryRun())
		return nil
	}
	// Execute full orchestration mode
	return project.Execute(ctx, r.cleanupRegistrar)
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
