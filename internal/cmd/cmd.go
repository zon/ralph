package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/requirement"
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

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// ConfigCmd defines the config subcommand group
type ConfigCmd struct {
	Git    ConfigGitCmd    `cmd:"" help:"Configure git credentials for remote execution"`
	Github ConfigGithubCmd `cmd:"" help:"Configure GitHub credentials for remote execution"`
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

// Run executes the config git command
func (c *ConfigGitCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Configuring Git credentials for Ralph remote execution...")
	fmt.Println()

	// Get the Kubernetes context to use
	kubeContext := c.Context
	if kubeContext == "" {
		currentCtx, err := k8s.GetCurrentContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured.", err)
		}
		kubeContext = currentCtx
		fmt.Printf("Using current Kubernetes context: %s\n", kubeContext)
	} else {
		fmt.Printf("Using Kubernetes context: %s\n", kubeContext)
	}

	// Get the namespace to use
	namespace := c.Namespace
	if namespace == "" {
		ns, err := k8s.GetNamespaceForContext(ctx, kubeContext)
		if err != nil {
			logger.Warningf("Failed to get namespace for context: %v", err)
		}
		if ns == "" {
			namespace = "default"
			fmt.Printf("Using namespace: %s (default)\n", namespace)
		} else {
			namespace = ns
			fmt.Printf("Using namespace: %s\n", namespace)
		}
	} else {
		fmt.Printf("Using namespace: %s\n", namespace)
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

	// Output the public key
	fmt.Println("Public SSH Key:")
	fmt.Println("===============")
	fmt.Println(publicKey)
	fmt.Println()

	// Output instructions
	fmt.Println("Next Steps:")
	fmt.Println("===========")
	fmt.Println("1. Copy the public key above")
	fmt.Println("2. Add it as a deploy key to your GitHub repository:")
	fmt.Println("   https://github.com/<owner>/<repo>/settings/keys/new")
	fmt.Println()
	fmt.Println("   OR add it to your GitHub account SSH keys:")
	fmt.Println("   https://github.com/settings/ssh/new")
	fmt.Println()
	fmt.Println("3. Make sure to enable 'Allow write access' if Ralph needs to push commits")
	fmt.Println()
	fmt.Printf("Configuration complete! The secret '%s' is ready for use in namespace '%s'.\n", k8s.GitSecretName, namespace)

	return nil
}

// Run executes the config github command
func (c *ConfigGithubCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Configuring GitHub credentials for Ralph remote execution...")
	fmt.Println()

	// Get the Kubernetes context to use
	kubeContext := c.Context
	if kubeContext == "" {
		currentCtx, err := k8s.GetCurrentContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured.", err)
		}
		kubeContext = currentCtx
		fmt.Printf("Using current Kubernetes context: %s\n", kubeContext)
	} else {
		fmt.Printf("Using Kubernetes context: %s\n", kubeContext)
	}

	// Get the namespace to use
	namespace := c.Namespace
	if namespace == "" {
		ns, err := k8s.GetNamespaceForContext(ctx, kubeContext)
		if err != nil {
			logger.Warningf("Failed to get namespace for context: %v", err)
		}
		if ns == "" {
			namespace = "default"
			fmt.Printf("Using namespace: %s (default)\n", namespace)
		} else {
			namespace = ns
			fmt.Printf("Using namespace: %s\n", namespace)
		}
	} else {
		fmt.Printf("Using namespace: %s\n", namespace)
	}

	fmt.Println()

	// Output link to GitHub token creation page
	fmt.Println("GitHub Personal Access Token Required")
	fmt.Println("======================================")
	fmt.Println()
	fmt.Println("Ralph needs a GitHub personal access token to create pull requests.")
	fmt.Println()
	fmt.Println("Create a new token with 'repo' scope at:")
	fmt.Println("  https://github.com/settings/tokens/new?description=Ralph%20Remote%20Execution&scopes=repo")
	fmt.Println()

	// Prompt for GitHub token
	fmt.Print("Enter your GitHub personal access token (input will be hidden): ")

	// Read token from stdin
	reader := bufio.NewReader(os.Stdin)
	tokenInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}

	token := strings.TrimSpace(tokenInput)
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
