package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/zon/ralph/internal/cmd"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/version"
)

// Date is set via ldflags during build
var Date = "unknown"

var cleanupManager = NewManager(output.NewClient(os.Stdout, os.Stderr, false))

func main() {
	cleanupManager.SetupSignalHandlers()

	defer cleanupManager.Cleanup()

	c := &cmd.Cmd{}
	c.SetVersion(version.Version(), Date)

	ctx := kong.Parse(c,
		kong.Name("ralph"),
		kong.Description("AI-powered development orchestration tool"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	err := ctx.Run()
	if errors.Is(err, orchestrationRun.ErrBlocked) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx.FatalIfErrorf(err)
}
