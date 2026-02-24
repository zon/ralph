package context

import "os"

// Context holds the execution context for ralph commands
type Context struct {
	ProjectFile   string
	MaxIterations int
	DryRun        bool
	Verbose       bool
	NoNotify      bool
	NoServices    bool
	Local         bool
	Follow        bool
	Notes         []string // Runtime notes to pass to the agent
	Instructions   string // Path to an instructions file that overrides the default instructions
	InstructionsMD string // Inline instructions content; overrides .ralph/instructions.md when set
	Repo           string // owner/repo override (e.g., "zon/ralph"); skips local git remote detection
	Branch         string // Branch override; skips local git GetCurrentBranch + sync check
}

// IsDryRun returns true if running in dry-run mode
func (c *Context) IsDryRun() bool {
	return c.DryRun
}

// IsVerbose returns true if verbose logging is enabled
func (c *Context) IsVerbose() bool {
	return c.Verbose
}

// ShouldNotify returns true if notifications should be sent
func (c *Context) ShouldNotify() bool {
	// Disable notifications if submitting a remote workflow without following
	if !c.Local && !c.Follow {
		return false
	}
	return !c.NoNotify
}

// ShouldStartServices returns true if services should be started
func (c *Context) ShouldStartServices() bool {
	return !c.NoServices
}

// IsLocal returns true if running locally instead of submitting to Argo Workflows
func (c *Context) IsLocal() bool {
	return c.Local
}

// ShouldFollow returns true if workflow logs should be followed after submission
func (c *Context) ShouldFollow() bool {
	return c.Follow
}

// IsWorkflowExecution returns true if running inside a workflow container
// This is detected via the RALPH_WORKFLOW_EXECUTION environment variable
func (c *Context) IsWorkflowExecution() bool {
	return os.Getenv("RALPH_WORKFLOW_EXECUTION") == "true"
}

// AddNote adds a runtime note to be passed to the agent
func (c *Context) AddNote(note string) {
	if c.Notes == nil {
		c.Notes = []string{}
	}
	c.Notes = append(c.Notes, note)
}

// HasNotes returns true if there are any notes
func (c *Context) HasNotes() bool {
	return len(c.Notes) > 0
}
