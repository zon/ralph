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
	Remote        bool
	Watch         bool
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
	// Disable notifications if remote mode is enabled without watch
	if c.Remote && !c.Watch {
		return false
	}
	return !c.NoNotify
}

// ShouldStartServices returns true if services should be started
func (c *Context) ShouldStartServices() bool {
	return !c.NoServices
}

// IsRemote returns true if running in remote execution mode
func (c *Context) IsRemote() bool {
	return c.Remote
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
