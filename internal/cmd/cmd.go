package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/requirement"
	"github.com/zon/ralph/internal/workflow"
)

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	// Subcommands
	Run         RunCmd         `cmd:"" default:"withargs" help:"Execute ralph with a project file (default command)"`
	Merge       MergeCmd       `cmd:"" help:"Submit an Argo workflow to merge a completed PR"`
	Config      ConfigCmd      `cmd:"" help:"Configure credentials for remote execution"`
	GithubToken GithubTokenCmd `cmd:"github-token" help:"Generate a GitHub App installation token"`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// ConfigCmd defines the config subcommand group
type ConfigCmd struct {
	Github        ConfigGithubCmd        `cmd:"" help:"Configure GitHub credentials for remote execution"`
	Opencode      ConfigOpencodeCmd      `cmd:"" help:"Configure OpenCode credentials for remote execution"`
	WebhookConfig ConfigWebhookConfigCmd `cmd:"webhook-config" help:"Provision webhook-config secret into Kubernetes"`
	WebhookSecret ConfigWebhookSecretCmd `cmd:"webhook-secret" help:"Provision webhook-secrets secret into Kubernetes"`
}

// GithubTokenCmd generates a GitHub App installation token
type GithubTokenCmd struct {
	Owner      string `help:"Repository owner (default: autodetected from git remote)" short:"o"`
	Repo       string `help:"Repository name (default: autodetected from git remote)" short:"r"`
	SecretsDir string `help:"Directory containing GitHub App credentials (default: /secrets/github)" default:"/secrets/github"`
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
	Repo          string `help:"Repository in owner/repo format, e.g. zon/ralph (overrides git remote detection)" optional:""`
	Branch        string `help:"Branch to use as the clone branch (overrides git current branch detection)" optional:""`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
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
		Local:         r.Local,
		Watch:         r.Watch,
		Instructions:  r.Instructions,
		Repo:          r.Repo,
		Branch:        r.Branch,
	}

	if r.Once {
		// Execute single iteration mode
		projectName := strings.TrimSuffix(filepath.Base(ctx.ProjectFile), filepath.Ext(ctx.ProjectFile))
		if err := requirement.Execute(ctx, r.cleanupRegistrar); err != nil {
			notify.Error(projectName, ctx.ShouldNotify() && !ctx.IsDryRun())
			return err
		}

		notify.Success(projectName, ctx.ShouldNotify() && !ctx.IsDryRun())
		return nil
	}
	// Execute full orchestration mode
	return project.Execute(ctx, r.cleanupRegistrar)
}
