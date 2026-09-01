package cmd

import (
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/orchestration/get"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

// GetCmd is the read-only `ralph get` command group.
type GetCmd struct {
	Complete   GetCompleteCmd   `cmd:"" help:"List the completion hashes recorded complete in the commit log"`
	Incomplete GetIncompleteCmd `cmd:"" help:"List the items that are not complete"`
}

// GetCompleteCmd reports which items are recorded complete. The project file
// is optional: without one the completion trailers are read from the commit
// log with no item array resolved.
type GetCompleteCmd struct {
	ProjectFile string `arg:"" optional:"" help:"Path to project file (optional)"`
	Items       string `help:"jq query selecting the item array" name:"items"`
	Base        string `help:"Base branch bounding the commit log" name:"base" short:"B"`
	Index       bool   `help:"Rejected: complete already emits hashes" name:"index"`
}

func (c *GetCompleteCmd) Run() error {
	if c.Index {
		return fmt.Errorf("--index is not applicable to get complete; it already emits completion hashes")
	}
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
	})
}

// GetIncompleteCmd reports which items of a project are left. A project file is
// required.
type GetIncompleteCmd struct {
	ProjectFile string `arg:"" help:"Path to project file"`
	Items       string `help:"jq query selecting the item array" name:"items"`
	Base        string `help:"Base branch bounding the commit log" name:"base" short:"B"`
	Index       bool   `help:"Emit indices instead of items" name:"index"`
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
	}, c.Index)
}
