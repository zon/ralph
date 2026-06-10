package cmd

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	// Subcommands
	Run            RunCmd            `cmd:"" default:"withargs" help:"Execute ralph with a project file (default command)"`
	Command        CommandCmd        `cmd:"" help:"Run a command in the ralph environment"`
	Merge          MergeCmd          `cmd:"" help:"Submit an Argo workflow to merge a completed PR"`
	Set            SetCmd            `cmd:"" help:"Configure ralph settings"`
	Workflow       WorkflowGroup     `cmd:"" help:"Run ralph workflow subcommands in a container"`
	Validate       ValidateCmd       `cmd:"" help:"Validate a project YAML file"`
	List           ListCmd           `cmd:"" help:"List Argo workflows"`
	Stop           StopCmd           `cmd:"" help:"Stop an Argo workflow"`
	Pass           PassCmd           `cmd:"" help:"Mark a project requirement as passing or failing"`

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
	Token   WorkflowTokenCmd   `cmd:"" help:"Generate a GitHub App installation token and configure git HTTPS authentication"`
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


