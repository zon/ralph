package cmd

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	// Subcommands
	Run        RunCmd        `cmd:"" default:"withargs" help:"Execute Ralph with a project file (default command)"`
	Loop       LoopCmd       `cmd:"" help:"Run AI iterations over a set of steps"`
	Command    CommandCmd    `cmd:"" help:"Run a command in a remote Ralph workflow"`
	Validate   ValidateCmd   `cmd:"" help:"Validate a project YAML file"`
	Complete   CompleteCmd   `cmd:"" help:"List the completion hashes recorded in the commit log of this branch"`
	Incomplete IncompleteCmd `cmd:"" help:"List project items not complete in this branch"`
	List       ListCmd       `cmd:"" help:"List Argo workflows"`
	Stop       StopCmd       `cmd:"" help:"Stop an Argo workflow"`
	Logs       LogsCmd       `cmd:"" help:"Get logs of an Argo workflow"`
	Setup      SetupCmd      `cmd:"" help:"Configure credentials for remote execution"`
	Workflow   WorkflowGroup `cmd:"" help:"Run Ralph workflow subcommands in a container"`
	Help       HelpGroup     `cmd:"" help:"Show help for Ralph topics"`

	version string `kong:"-"`
	date    string `kong:"-"`
}

// WorkflowGroup defines the workflow subcommand group
type WorkflowGroup struct {
	Run     WorkflowRunCmd     `cmd:"" help:"Invoked by workflows to execute a project"`
	Loop    WorkflowLoopCmd    `cmd:"" help:"Invoked by workflows to execute a loop"`
	Command WorkflowCommandCmd `cmd:"" help:"Invoked by workflows to execute a command"`
	Token   WorkflowTokenCmd   `cmd:"" help:"Invoked by workflows to configure git and gh auth"`
}

// HelpGroup defines the help subcommand group
type HelpGroup struct {
	Config HelpConfigCmd `cmd:"" help:"Display the configuration reference"`
}

// SetVersion sets the version information
func (c *Cmd) SetVersion(version, date string) {
	c.version = version
	c.date = date
	c.Run.version = version
	c.Run.date = date
}
