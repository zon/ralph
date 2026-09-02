package cmd

import (
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/orchestration/get"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

// GetCmd is the read-only `ralph get` command group.
type GetCmd struct {
	Complete   GetCompleteCmd   `cmd:"" help:"List the completion hashes recorded in the commit log of this branch"`
	Incomplete GetIncompleteCmd `cmd:"" help:"List project items not complete in this branch"`
}

// GetCompleteCmd reports which items are recorded complete. The project file
// is optional: without one the completion trailers are read from the commit
// log with no item array resolved.
type GetCompleteCmd struct {
	ProjectFile string `arg:"" optional:"" help:"Path to project file (optional)"`
	Items       string `help:"jq query selecting the project item list (default: .)" name:"items" short:"i"`
	Base        string `help:"Base branch bounding the commit log" name:"base" short:"B"`
	JSON        bool   `help:"Print the hashes as a JSON array" name:"json"`
}

func (c *GetCompleteCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	cmd := get.NewCmd(project.NewClient(git.NewClient(ctx), ctx.Output()), os.Stdout)
	return cmd.Complete(cfg, get.Flags{
		ProjectFile: c.ProjectFile,
		Items:       c.Items,
		Base:        c.Base,
		JSON:        c.JSON,
	})
}

// GetIncompleteCmd reports which items of a project are left. A project file is
// required.
type GetIncompleteCmd struct {
	ProjectFile string `arg:"" help:"Path to project file"`
	Items       string `help:"jq query selecting the project item list (default: .)" name:"items" short:"i"`
	Base        string `help:"Base branch bounding the commit log" name:"base" short:"B"`
}

func (c *GetIncompleteCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	cmd := get.NewCmd(project.NewClient(git.NewClient(ctx), ctx.Output()), os.Stdout)
	return cmd.Incomplete(cfg, get.Flags{
		ProjectFile: c.ProjectFile,
		Items:       c.Items,
		Base:        c.Base,
	})
}
