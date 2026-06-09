package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/zon/ralph/internal/cleanup"
	"github.com/zon/ralph/internal/cmd"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/version"
)

// Date is set via ldflags during build
var Date = "unknown"

var cleanupManager = cleanup.NewManager(output.NewClient(os.Stdout, os.Stderr, false))

func main() {
	cleanupManager.SetupSignalHandlers()

	defer cleanupManager.Cleanup()

	c := &cmd.Cmd{}
	c.SetVersion(version.Version(), Date)
	c.SetCleanupRegistrar(cleanupManager.RegisterCleanup)

	ctx := kong.Parse(c,
		kong.Name("ralph"),
		kong.Description("AI-powered development orchestration tool"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
