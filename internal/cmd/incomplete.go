package cmd

import (
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/orchestration/completion"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

// IncompleteCmd reports which items of a project are left. A project file is
// required.
type IncompleteCmd struct {
	ProjectFile string `arg:"" help:"Path to project file"`
	Items       string `help:"jq query selecting the project item list (default: .)" name:"items" short:"i"`
	Base        string `help:"Base branch bounding the commit log" name:"base" short:"B"`
}

func (c *IncompleteCmd) Run() error {
	ctx := createExecutionContext()
	// The JSON array on stdout is the command's only stdout contract. Route
	// diagnostics, such as project reconciliation warnings, to stderr so
	// scripts that parse the output never receive non-JSON text.
	ctx.SetOutput(output.NewClient(os.Stderr, os.Stderr, false))
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	cmd := completion.NewCmd(project.NewClient(git.NewClient(ctx), ctx.Output()), os.Stdout)
	return cmd.Incomplete(cfg, completion.Flags{
		ProjectFile: c.ProjectFile,
		Items:       c.Items,
		Base:        c.Base,
	})
}
