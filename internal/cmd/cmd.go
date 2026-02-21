package cmd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
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
	"github.com/zon/ralph/internal/webhook"
	"github.com/zon/ralph/internal/workflow"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	// Subcommands
	Run    RunCmd    `cmd:"" default:"withargs" help:"Execute ralph with a project file (default command)"`
	Merge  MergeCmd  `cmd:"" help:"Submit an Argo workflow to merge a completed PR"`
	Config ConfigCmd `cmd:"" help:"Configure credentials for remote execution"`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// MergeCmd is the command for submitting a merge workflow for a completed PR
type MergeCmd struct {
	ProjectFile string `arg:"" help:"Path to project YAML file" type:"path"`
	Branch      string `arg:"" help:"PR branch name to merge"`
	DryRun      bool   `help:"Simulate execution without making changes" default:"false"`
	Verbose     bool   `help:"Enable verbose logging" default:"false"`
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
	Local         bool   `help:"Run on this machine instead of in Argo Workflows" default:"false"`
	Watch         bool   `help:"Watch workflow execution (only applicable without --local)" default:"false"`
	ShowVersion   bool   `help:"Show version information" short:"v" name:"version"`
	Instructions  string `help:"Path to an instructions file that overrides the default agent instructions" type:"path" optional:""`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// ConfigCmd defines the config subcommand group
type ConfigCmd struct {
	Git           ConfigGitCmd           `cmd:"" help:"Configure git credentials for remote execution"`
	Github        ConfigGithubCmd        `cmd:"" help:"Configure GitHub credentials for remote execution"`
	Opencode      ConfigOpencodeCmd      `cmd:"" help:"Configure OpenCode credentials for remote execution"`
	WebhookConfig ConfigWebhookConfigCmd `cmd:"webhook-config" help:"Provision webhook-config secret into Kubernetes"`
	WebhookSecret ConfigWebhookSecretCmd `cmd:"webhook-secret" help:"Provision webhook-secrets secret into Kubernetes"`
}

// ConfigWebhookConfigCmd provisions the webhook-config Kubernetes secret
type ConfigWebhookConfigCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
	Config    string `help:"Path to a partial AppConfig YAML file to use as a starting point" type:"path" optional:""`
	DryRun    bool   `help:"Simulate execution without making changes" default:"false"`
}

// ConfigWebhookSecretCmd provisions the webhook-secrets Kubernetes secret
type ConfigWebhookSecretCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
	DryRun    bool   `help:"Simulate execution without making changes" default:"false"`
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

// Run executes the merge command (implements kong.Run interface)
func (m *MergeCmd) Run() error {
	if m.ProjectFile == "" {
		return fmt.Errorf("project file required (see --help)")
	}
	if m.Branch == "" {
		return fmt.Errorf("PR branch name required (see --help)")
	}

	// Load ralph config for workflow submission
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load ralph config: %w", err)
	}

	// Generate the merge workflow
	workflowYAML, err := workflow.GenerateMergeWorkflow(m.ProjectFile, m.Branch)
	if err != nil {
		return fmt.Errorf("failed to generate merge workflow: %w", err)
	}

	if m.DryRun {
		logger.Infof("Dry run: would submit merge workflow for branch %s", m.Branch)
		if m.Verbose {
			fmt.Println(workflowYAML)
		}
		return nil
	}

	// Submit the workflow (does not wait for completion)
	ctx := &execcontext.Context{
		ProjectFile: m.ProjectFile,
		DryRun:      m.DryRun,
		Verbose:     m.Verbose,
	}
	workflowName, err := workflow.SubmitWorkflow(ctx, workflowYAML, ralphConfig)
	if err != nil {
		return fmt.Errorf("failed to submit merge workflow: %w", err)
	}

	logger.Successf("Merge workflow submitted: %s", workflowName)
	return nil
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
	if r.Watch && r.Local {
		return fmt.Errorf("--watch flag is not applicable with --local flag")
	}

	if r.Local && r.Once {
		return fmt.Errorf("--local flag is incompatible with --once flag")
	}

	// Create execution context
	ctx := &execcontext.Context{
		ProjectFile:   r.ProjectFile,
		MaxIterations: r.MaxIterations,
		DryRun:        r.DryRun,
		Verbose:       r.Verbose,
		NoNotify:      r.NoNotify,
		NoServices:    r.NoServices,
		Remote:        !r.Local,
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

// webhookConfigSecretName is the name of the Kubernetes secret for the webhook app config
const webhookConfigSecretName = "webhook-config"

// webhookSecretsSecretName is the name of the Kubernetes secret for the webhook secrets
const webhookSecretsSecretName = "webhook-secrets"

// WebhookConfigResult holds the result of building an AppConfig for dry-run inspection
type WebhookConfigResult struct {
	AppConfig webhook.AppConfig
	YAML      string
}

// buildWebhookAppConfig builds an AppConfig with defaults filled in.
// partialConfig is an optional starting point (may be nil).
// repoName and repoOwner are the detected GitHub repo details.
// model is the AI model from .ralph/config.yaml.
// ralphUsername is the authenticated gh CLI user.
func buildWebhookAppConfig(partialConfig *webhook.AppConfig, repoName, repoOwner, model, ralphUsername string) webhook.AppConfig {
	var cfg webhook.AppConfig

	// Start with partial config if provided
	if partialConfig != nil {
		cfg = *partialConfig
	}

	// Fill in port default
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	// Fill in model from .ralph/config.yaml if not set
	if cfg.Model == "" {
		cfg.Model = model
	}

	// Fill in ralphUsername from gh CLI if not set
	if cfg.RalphUsername == "" {
		cfg.RalphUsername = ralphUsername
	}

	// Auto-add repo if detected and not already present
	if repoName != "" && repoOwner != "" {
		found := false
		for _, r := range cfg.Repos {
			if r.Owner == repoOwner && r.Name == repoName {
				found = true
				break
			}
		}
		if !found {
			cfg.Repos = append(cfg.Repos, webhook.RepoConfig{
				Owner:     repoOwner,
				Name:      repoName,
				ClonePath: fmt.Sprintf("/repos/%s", repoName),
			})
		} else {
			// Fill in clonePath default for existing entries
			for i, r := range cfg.Repos {
				if r.Owner == repoOwner && r.Name == repoName && r.ClonePath == "" {
					cfg.Repos[i].ClonePath = fmt.Sprintf("/repos/%s", repoName)
				}
			}
		}
	}

	return cfg
}

// Run executes the config webhook-config command
func (c *ConfigWebhookConfigCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Provisioning webhook-config secret...")
	fmt.Println()

	// Determine Kubernetes context (namespace defaults to ralph-webhook via struct tag)
	var kubeContext string
	if c.Context != "" {
		kubeContext = c.Context
	} else {
		currentCtx, err := k8s.GetCurrentContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current Kubernetes context: %w", err)
		}
		kubeContext = currentCtx
	}

	namespace := c.Namespace

	// Load optional partial config
	var partialConfig *webhook.AppConfig
	if c.Config != "" {
		loaded, err := webhook.LoadAppConfig(c.Config)
		if err != nil {
			return fmt.Errorf("failed to load partial config: %w", err)
		}
		partialConfig = loaded
	}

	// Auto-detect repo from git remote
	repoName, repoOwner, err := k8s.GetGitHubRepo(ctx)
	if err != nil {
		logger.Warningf("Failed to detect GitHub repository: %v (skipping repo auto-detection)", err)
		repoName = ""
		repoOwner = ""
	}

	// Load model from .ralph/config.yaml
	model := ""
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		logger.Warningf("Failed to load .ralph/config.yaml: %v", err)
	} else if ralphConfig != nil {
		model = ralphConfig.Model
	}

	// Get authenticated gh CLI user for ralphUsername default
	ralphUsername := ""
	ghUserCmd := exec.CommandContext(ctx, "gh", "api", "user", "--jq", ".login")
	if out, err := ghUserCmd.Output(); err == nil {
		ralphUsername = strings.TrimSpace(string(out))
	} else {
		logger.Warningf("Failed to get GitHub username from gh CLI: %v", err)
	}

	// Build AppConfig with defaults filled in
	appCfg := buildWebhookAppConfig(partialConfig, repoName, repoOwner, model, ralphUsername)

	// Serialize to YAML
	cfgBytes, err := yaml.Marshal(appCfg)
	if err != nil {
		return fmt.Errorf("failed to serialize AppConfig to YAML: %w", err)
	}

	cfgYAML := string(cfgBytes)
	fmt.Printf("AppConfig YAML:\n%s\n", cfgYAML)

	if c.DryRun {
		fmt.Printf("Dry run: would write secret '%s' in namespace '%s' (context: %s)\n", webhookConfigSecretName, namespace, kubeContext)
		return nil
	}

	// Write to Kubernetes secret
	secretData := map[string]string{
		"config.yaml": cfgYAML,
	}

	if err := k8s.CreateOrUpdateSecret(ctx, webhookConfigSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret '%s': %w", webhookConfigSecretName, err)
	}

	fmt.Printf("Secret '%s' created/updated in namespace '%s'\n", webhookConfigSecretName, namespace)
	return nil
}

// webhookIngressHostname is the default ingress hostname used to match webhooks on GitHub
const webhookIngressHostname = "ralph.wurbs.chat"

// generateWebhookSecret generates a cryptographically random webhook secret (32 bytes, hex-encoded)
func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// readWebhookConfigFromK8s reads the webhook-config secret from Kubernetes and returns
// the AppConfig it contains.
func readWebhookConfigFromK8s(ctx context.Context, namespace, kubeContext string) (*webhook.AppConfig, error) {
	// kubectl get secret webhook-config -n <ns> --context <ctx> -o jsonpath='{.data.config\.yaml}'
	args := []string{
		"get", "secret", webhookConfigSecretName,
		"-n", namespace,
		"-o", `jsonpath={.data.config\.yaml}`,
	}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to read secret '%s' from namespace '%s': %w (stderr: %s)",
			webhookConfigSecretName, namespace, err, stderr.String())
	}

	// The value is base64-encoded by Kubernetes
	encoded := strings.TrimSpace(stdout.String())
	if encoded == "" {
		return nil, fmt.Errorf("secret '%s' exists but config.yaml key is empty", webhookConfigSecretName)
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode config.yaml from secret: %w", err)
	}

	var appCfg webhook.AppConfig
	if err := yaml.Unmarshal(decoded, &appCfg); err != nil {
		return nil, fmt.Errorf("failed to parse AppConfig YAML from secret: %w", err)
	}

	return &appCfg, nil
}

// registerGitHubWebhook registers (or updates) a webhook on GitHub for the given repo
// using the provided secret. It looks for an existing webhook matching the ingress hostname
// and updates it; otherwise it creates a new one.
func registerGitHubWebhook(ctx context.Context, owner, repo, webhookURL, secret string) error {
	// List hooks and find one matching our URL
	listArgs := []string{
		"api",
		fmt.Sprintf("repos/%s/%s/hooks", owner, repo),
		"--jq", fmt.Sprintf(`.[] | select(.config.url | contains("%s")) | .id`, webhookURL),
	}
	listCmdFull := exec.CommandContext(ctx, "gh", listArgs...)
	var listOut, listErr bytes.Buffer
	listCmdFull.Stdout = &listOut
	listCmdFull.Stderr = &listErr

	var existingID string
	if err := listCmdFull.Run(); err == nil {
		existingID = strings.TrimSpace(listOut.String())
	}

	// Build the webhook payload
	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"push", "pull_request", "pull_request_review", "issue_comment"},
		"config": map[string]string{
			"url":          webhookURL,
			"content_type": "json",
			"secret":       secret,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	if existingID != "" && existingID != "null" {
		// Update existing webhook
		updateCmd := exec.CommandContext(ctx, "gh", "api",
			fmt.Sprintf("repos/%s/%s/hooks/%s", owner, repo, existingID),
			"--method", "PATCH",
			"--input", "-",
		)
		updateCmd.Stdin = bytes.NewReader(payloadBytes)
		var updateErr bytes.Buffer
		updateCmd.Stderr = &updateErr
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to update webhook for %s/%s: %w (stderr: %s)",
				owner, repo, err, updateErr.String())
		}
		return nil
	}

	// Create new webhook
	createCmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/%s/hooks", owner, repo),
		"--method", "POST",
		"--input", "-",
	)
	createCmd.Stdin = bytes.NewReader(payloadBytes)
	var createErr bytes.Buffer
	createCmd.Stderr = &createErr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create webhook for %s/%s: %w (stderr: %s)",
			owner, repo, err, createErr.String())
	}

	return nil
}

// WebhookSecretsResult holds the result of building Secrets for dry-run inspection
type WebhookSecretsResult struct {
	Secrets webhook.Secrets
	YAML    string
}

// buildWebhookSecrets generates cryptographically random webhook secrets for each repo
// in the provided AppConfig. It uses the provided secretGenerator function to allow
// testing with predictable values.
func buildWebhookSecrets(appCfg *webhook.AppConfig, secretGenerator func() (string, error)) (*webhook.Secrets, error) {
	secrets := &webhook.Secrets{}

	for _, repo := range appCfg.Repos {
		secret, err := secretGenerator()
		if err != nil {
			return nil, fmt.Errorf("failed to generate secret for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		secrets.Repos = append(secrets.Repos, webhook.RepoSecret{
			Owner:         repo.Owner,
			Name:          repo.Name,
			WebhookSecret: secret,
		})
	}

	return secrets, nil
}

// Run executes the config webhook-secret command
func (c *ConfigWebhookSecretCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Provisioning webhook-secrets secret...")
	fmt.Println()

	// Determine Kubernetes context (namespace defaults to ralph-webhook via struct tag)
	var kubeContext string
	if c.Context != "" {
		kubeContext = c.Context
	} else {
		currentCtx, err := k8s.GetCurrentContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current Kubernetes context: %w", err)
		}
		kubeContext = currentCtx
	}

	namespace := c.Namespace

	// Read repo list from webhook-config secret
	fmt.Printf("Reading repo list from secret '%s' in namespace '%s'...\n", webhookConfigSecretName, namespace)
	appCfg, err := readWebhookConfigFromK8s(ctx, namespace, kubeContext)
	if err != nil {
		return fmt.Errorf("failed to read webhook-config: %w\n\nRun 'ralph config webhook-config' first to create the webhook-config secret.", err)
	}

	if len(appCfg.Repos) == 0 {
		return fmt.Errorf("no repos found in webhook-config secret — add repos first via 'ralph config webhook-config'")
	}

	fmt.Printf("Found %d repo(s) in webhook-config\n\n", len(appCfg.Repos))

	// Generate webhook secrets for each repo
	secrets, err := buildWebhookSecrets(appCfg, generateWebhookSecret)
	if err != nil {
		return fmt.Errorf("failed to generate webhook secrets: %w", err)
	}

	// Serialize to YAML
	secretsBytes, err := yaml.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("failed to serialize Secrets to YAML: %w", err)
	}

	secretsYAML := string(secretsBytes)
	fmt.Printf("Generated webhook secrets for %d repo(s)\n\n", len(secrets.Repos))

	if c.DryRun {
		fmt.Printf("Dry run: would write secret '%s' in namespace '%s' (context: %s)\n", webhookSecretsSecretName, namespace, kubeContext)
		for _, rs := range secrets.Repos {
			fmt.Printf("  - %s/%s: (secret generated, not shown in dry-run)\n", rs.Owner, rs.Name)
		}
		return nil
	}

	// Register webhooks on GitHub for each repo
	webhookURL := fmt.Sprintf("https://%s/webhook", webhookIngressHostname)
	fmt.Printf("Registering webhooks at %s...\n", webhookURL)
	for _, rs := range secrets.Repos {
		fmt.Printf("  Registering webhook for %s/%s...\n", rs.Owner, rs.Name)
		if err := registerGitHubWebhook(ctx, rs.Owner, rs.Name, webhookURL, rs.WebhookSecret); err != nil {
			logger.Warningf("Failed to register webhook for %s/%s: %v", rs.Owner, rs.Name, err)
			fmt.Printf("  ⚠ Failed to register webhook for %s/%s: %v\n", rs.Owner, rs.Name, err)
		} else {
			fmt.Printf("  ✓ Webhook registered for %s/%s\n", rs.Owner, rs.Name)
		}
	}
	fmt.Println()

	// Write to Kubernetes secret
	secretData := map[string]string{
		"secrets.yaml": secretsYAML,
	}

	if err := k8s.CreateOrUpdateSecret(ctx, webhookSecretsSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret '%s': %w", webhookSecretsSecretName, err)
	}

	fmt.Printf("Secret '%s' created/updated in namespace '%s'\n", webhookSecretsSecretName, namespace)
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
