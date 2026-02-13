package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/zon/ralph/internal/cleanup"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/once"
	"github.com/zon/ralph/internal/run"
)

// Version information
var (
	Version = "0.1.0"
	Date    = "unknown"
)

// Global cleanup manager instance
var cleanupManager = cleanup.NewManager()

// CLI defines the command-line interface structure
type CLI struct {
	Run     RunCmd     `cmd:"" help:"Full orchestration: create branch, iterate development, submit PR"`
	Once    OnceCmd    `cmd:"" help:"Single development iteration"`
	Version VersionCmd `cmd:"" help:"Show version information"`
}

// RunCmd represents the 'ralph run' command
type RunCmd struct {
	ProjectFile   string `arg:"" help:"Path to project YAML file" type:"path"`
	MaxIterations int    `help:"Maximum number of development iterations" default:"10"`
	DryRun        bool   `help:"Simulate execution without making changes" default:"false"`
	NoNotify      bool   `help:"Disable desktop notifications" default:"false"`
	Verbose       bool   `help:"Enable verbose logging" default:"false"`
}

// Run executes the run command
func (r *RunCmd) Run(ctx *kong.Context) error {
	// Note: NoServices is not applicable for run command, always false
	execCtx := context.NewContext(r.DryRun, r.Verbose, r.NoNotify, false)
	return run.Execute(execCtx, r.ProjectFile, r.MaxIterations, cleanupManager.RegisterCleanup)
}

// OnceCmd represents the 'ralph once' command
type OnceCmd struct {
	ProjectFile string `arg:"" help:"Path to project YAML file" type:"path"`
	DryRun      bool   `help:"Simulate execution without making changes" default:"false"`
	NoNotify    bool   `help:"Disable desktop notifications" default:"false"`
	NoServices  bool   `help:"Skip service startup" default:"false"`
	Verbose     bool   `help:"Enable verbose logging" default:"false"`
}

// Run executes the once command
func (o *OnceCmd) Run(ctx *kong.Context) error {
	execCtx := context.NewContext(o.DryRun, o.Verbose, o.NoNotify, o.NoServices)
	return once.Execute(execCtx, o.ProjectFile, cleanupManager.RegisterCleanup)
}

// VersionCmd represents the 'ralph version' command
type VersionCmd struct{}

// Run executes the version command
func (v *VersionCmd) Run(ctx *kong.Context) error {
	if Date != "unknown" {
		fmt.Printf("ralph version %s (%s)\n", Version, Date)
	} else {
		fmt.Printf("ralph version %s\n", Version)
	}
	return nil
}

func main() {
	// Set up signal handlers for graceful shutdown
	cleanupManager.SetupSignalHandlers()

	// Ensure cleanup happens on normal exit
	defer cleanupManager.Cleanup()

	cli := &CLI{}

	ctx := kong.Parse(cli,
		kong.Name("ralph"),
		kong.Description("AI-powered development orchestration tool"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	// Execute the selected command
	err := ctx.Run(ctx)
	ctx.FatalIfErrorf(err)
}
