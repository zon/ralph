package cmd

import (
	"github.com/zon/ralph/internal/orchestration/pass"
)

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	// Subcommands
	Run            RunCmd            `cmd:"" default:"withargs" help:"Execute ralph with a project file (default command)"`
	Command        CommandCmd        `cmd:"" help:"Run a command in the ralph environment"`
	Merge          MergeCmd          `cmd:"" help:"Submit an Argo workflow to merge a completed PR"`
	Config         ConfigCmd         `cmd:"" help:"Configure credentials for remote execution"`
	SetGithubToken GithubTokenCmd    `cmd:"" help:"Generate a GitHub App installation token and configure git HTTPS authentication"`
	Set            SetCmd            `cmd:"" help:"Configure ralph settings"`
	Workflow       WorkflowGroup     `cmd:"" help:"Run ralph workflow subcommands in a container"`
	Validate       ValidateCmd       `cmd:"" help:"Validate a project YAML file"`
	List           ListCmd           `cmd:"" help:"List Argo workflows"`
	Stop           StopCmd           `cmd:"" help:"Stop an Argo workflow"`
	Pass           pass.PassCmd      `cmd:"" help:"Mark a project requirement as passing or failing"`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// WorkflowGroup defines the workflow subcommand group
type WorkflowGroup struct {
	Run     WorkflowRunCmd     `cmd:"" help:"Run a project via the workflow engine"`
	Comment WorkflowCommentCmd `cmd:"" help:"Run a comment-triggered workflow iteration"`
	Merge   WorkflowMergeCmd   `cmd:"" help:"Merge a completed PR via workflow"`
	Command WorkflowCommandCmd `cmd:"" help:"Run an arbitrary command via workflow"`
}

// ConfigCmd defines the config subcommand group
type ConfigCmd struct {
	Github        ConfigGithubCmd        `cmd:"" help:"Configure GitHub credentials for remote execution"`
	Opencode      ConfigOpencodeCmd      `cmd:"" help:"Configure OpenCode credentials for remote execution"`
	Pulumi        ConfigPulumiCmd        `cmd:"" help:"Configure Pulumi credentials for remote execution"`
	WebhookConfig ConfigWebhookConfigCmd `cmd:"" name:"webhook" help:"Provision webhook-config secret into Kubernetes"`
	WebhookSecret ConfigWebhookSecretCmd `cmd:"" help:"Provision webhook-secrets secret into Kubernetes"`
}

// GithubTokenCmd generates a GitHub App installation token
type GithubTokenCmd struct {
	Owner      string `help:"Repository owner (default: autodetected from git remote)" short:"o"`
	Repo       string `help:"Repository name (default: autodetected from git remote)" short:"r"`
	SecretsDir string `help:"Directory containing GitHub App credentials (default: /secrets/github)" default:"/secrets/github"`
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
	c.Command.cleanupRegistrar = cleanupRegistrar
	c.Merge.cleanupRegistrar = cleanupRegistrar
	c.Workflow.Run.cleanupRegistrar = cleanupRegistrar
	c.Workflow.Comment.cleanupRegistrar = cleanupRegistrar
	c.Workflow.Merge.cleanupRegistrar = cleanupRegistrar
	c.Workflow.Command.cleanupRegistrar = cleanupRegistrar
}


