package cmd

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	// Subcommands
	Run        RunCmd        `cmd:"" default:"withargs" help:"Execute Ralph with a project file (default command)"`
	Command    CommandCmd    `cmd:"" help:"Run a command in a remote Ralph workflow"`
	Setup      SetupCmd      `cmd:"" help:"Configure credentials for remote execution"`
	Complete   CompleteCmd   `cmd:"" help:"List the completion hashes recorded in the commit log of this branch"`
	Incomplete IncompleteCmd `cmd:"" help:"List project items not complete in this branch"`
	Workflow   WorkflowGroup `cmd:"" help:"Run Ralph workflow subcommands in a container"`
	Validate   ValidateCmd   `cmd:"" help:"Validate a project YAML file"`
	Config     ConfigCmd     `cmd:"" help:"Display the configuration reference"`
	List       ListCmd       `cmd:"" help:"List Argo workflows"`
	Stop       StopCmd       `cmd:"" help:"Stop an Argo workflow"`
	Logs       LogsCmd       `cmd:"" help:"Get logs of an Argo workflow"`
	Loop       LoopCmd       `cmd:"" help:"Run AI iterations over a set of steps"`

	version string `kong:"-"`
	date    string `kong:"-"`
}

// WorkflowGroup defines the workflow subcommand group
type WorkflowGroup struct {
	Run     WorkflowRunCmd     `cmd:"" help:"Invoked by workflows to execute a project"`
	Command WorkflowCommandCmd `cmd:"" help:"Invoked by workflows to execute a command"`
	Token   WorkflowTokenCmd   `cmd:"" help:"Invoked by workflows to configure git and gh auth"`
	Loop    WorkflowLoopCmd    `cmd:"" help:"Invoked by workflows to execute a loop"`
}

// SetVersion sets the version information
func (c *Cmd) SetVersion(version, date string) {
	c.version = version
	c.date = date
	c.Run.version = version
	c.Run.date = date
}
