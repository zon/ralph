package context

// Context holds the execution context for ralph commands
type Context struct {
	ProjectFile   string
	MaxIterations int
	DryRun        bool
	Verbose       bool
	NoNotify      bool
	NoServices    bool
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
	return !c.NoNotify
}

// ShouldStartServices returns true if services should be started
func (c *Context) ShouldStartServices() bool {
	return !c.NoServices
}
