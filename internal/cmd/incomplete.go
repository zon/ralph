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
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
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
