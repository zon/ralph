package main

import (
	"github.com/alecthomas/kong"
	"github.com/zon/ralph/internal/cleanup"
	"github.com/zon/ralph/internal/cmd"
)

// Version information
var (
	Version = "0.1.0"
	Date    = "unknown"
)

// Global cleanup manager instance
var cleanupManager = cleanup.NewManager()

func main() {
	// Set up signal handlers for graceful shutdown
	cleanupManager.SetupSignalHandlers()

	// Ensure cleanup happens on normal exit
	defer cleanupManager.Cleanup()

	c := &cmd.Cmd{}
	c.SetVersion(Version, Date)
	c.SetCleanupRegistrar(cleanupManager.RegisterCleanup)

	ctx := kong.Parse(c,
		kong.Name("ralph"),
		kong.Description("AI-powered development orchestration tool"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	// Execute the selected command
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
