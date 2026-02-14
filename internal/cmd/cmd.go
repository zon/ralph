package cmd

import (
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/once"
	"github.com/zon/ralph/internal/run"
)

// Cmd defines the command-line arguments and execution context
type Cmd struct {
	WorkingDir    string `help:"Working directory to run ralph in" type:"path" short:"C"`
	ProjectFile   string `arg:"" optional:"" help:"Path to project YAML file" type:"path"`
	Once          bool   `help:"Single development iteration mode" default:"false"`
	MaxIterations int    `help:"Maximum number of development iterations (not applicable with --once)" default:"10"`
	DryRun        bool   `help:"Simulate execution without making changes" default:"false"`
	NoNotify      bool   `help:"Disable desktop notifications" default:"false"`
	NoServices    bool   `help:"Skip service startup" default:"false"`
	Verbose       bool   `help:"Enable verbose logging" default:"false"`
	ShowVersion   bool   `help:"Show version information" short:"v" name:"version"`

	version          string       `kong:"-"`
	date             string       `kong:"-"`
	cleanupRegistrar func(func()) `kong:"-"`
}

// SetVersion sets the version information
func (c *Cmd) SetVersion(version, date string) {
	c.version = version
	c.date = date
}

// SetCleanupRegistrar sets the cleanup registrar function
func (c *Cmd) SetCleanupRegistrar(cleanupRegistrar func(func())) {
	c.cleanupRegistrar = cleanupRegistrar
}

// Run executes the main command (implements kong.Run interface)
func (c *Cmd) Run() error {
	// Handle version flag
	if c.ShowVersion {
		if c.date != "unknown" {
			fmt.Printf("ralph version %s (%s)\n", c.version, c.date)
		} else {
			fmt.Printf("ralph version %s\n", c.version)
		}
		return nil
	}

	// Change working directory if specified
	if c.WorkingDir != "" {
		if err := os.Chdir(c.WorkingDir); err != nil {
			return fmt.Errorf("failed to change to working directory %s: %w", c.WorkingDir, err)
		}
	}

	// Validate required fields
	if c.ProjectFile == "" {
		return fmt.Errorf("project file required (see --help)")
	}

	// Create execution context
	ctx := &execcontext.Context{
		ProjectFile:   c.ProjectFile,
		MaxIterations: c.MaxIterations,
		DryRun:        c.DryRun,
		Verbose:       c.Verbose,
		NoNotify:      c.NoNotify,
		NoServices:    c.NoServices,
	}

	if c.Once {
		// Execute single iteration mode
		// Load project for notification
		project, err := config.LoadProject(ctx.ProjectFile)
		if err != nil {
			return fmt.Errorf("failed to load project: %w", err)
		}

		if err := once.Execute(ctx, c.cleanupRegistrar); err != nil {
			notify.Error(project.Name, ctx.ShouldNotify() && !ctx.IsDryRun())
			return err
		}

		notify.Success(project.Name, ctx.ShouldNotify() && !ctx.IsDryRun())
		return nil
	}
	// Execute full orchestration mode
	return run.Execute(ctx, c.cleanupRegistrar)
}
