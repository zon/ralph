package context

// Context holds the execution context for ralph commands
type Context struct {
	DryRun     bool
	Verbose    bool
	NoNotify   bool
	NoServices bool // Only applicable for 'once' command
}

// NewContext creates a new execution context
func NewContext(dryRun, verbose, noNotify, noServices bool) *Context {
	return &Context{
		DryRun:     dryRun,
		Verbose:    verbose,
		NoNotify:   noNotify,
		NoServices: noServices,
	}
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
