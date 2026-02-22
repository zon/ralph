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
	Watch         bool
	Notes         []string // Runtime notes to pass to the agent
	Instructions  string   // Path to an instructions file that overrides the default instructions
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
	// Disable notifications if submitting a remote workflow without watching
	if !c.Local && !c.Watch {
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

// ShouldWatch returns true if workflow execution should be watched
func (c *Context) ShouldWatch() bool {
	return c.Watch
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
