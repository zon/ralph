package cmd

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	// Subcommands
	Run      RunCmd        `cmd:"" default:"withargs" help:"Execute ralph with a project file (default command)"`
	Command  CommandCmd    `cmd:"" help:"Run a command in a remote Ralph workflow"`
	Set      SetCmd        `cmd:"" help:"Configure ralph settings"`
	Get      GetCmd        `cmd:"" help:"Report which items are complete and which are left"`
	Workflow WorkflowGroup `cmd:"" help:"Run ralph workflow subcommands in a container"`
	Validate ValidateCmd   `cmd:"" help:"Validate a project YAML file"`
	List     ListCmd       `cmd:"" help:"List Argo workflows"`
	Stop     StopCmd       `cmd:"" help:"Stop an Argo workflow"`
	Logs     LogsCmd       `cmd:"" help:"Get logs of an Argo workflow"`
	Loop     LoopCmd       `cmd:"" help:"Run AI iterations over a set of steps"`

	version string `kong:"-"`
	date    string `kong:"-"`
}

// WorkflowGroup defines the workflow subcommand group
type WorkflowGroup struct {
	Run     WorkflowRunCmd     `cmd:"" help:"Run a project via the workflow engine"`
	Comment WorkflowCommentCmd `cmd:"" help:"Run a comment-triggered workflow iteration"`
	Command WorkflowCommandCmd `cmd:"" help:"Run an arbitrary command via workflow"`
	Token   WorkflowTokenCmd   `cmd:"" help:"Generate a GitHub App installation token and configure git HTTPS authentication"`
	Loop    WorkflowLoopCmd    `cmd:"" help:"Run a loop via the workflow engine"`
}

// SetVersion sets the version information
func (c *Cmd) SetVersion(version, date string) {
	c.version = version
	c.date = date
	c.Run.version = version
	c.Run.date = date
}
