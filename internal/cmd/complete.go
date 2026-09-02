package cmd

import (
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/orchestration/completion"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

// CompleteCmd reports which items are recorded complete. The project file is
// optional: without one the completion trailers are read from the commit log
// with no item array resolved.
type CompleteCmd struct {
	ProjectFile string `arg:"" optional:"" help:"Path to project file (optional)"`
	Items       string `help:"jq query selecting the project item list (default: .)" name:"items" short:"i"`
	Base        string `help:"Base branch bounding the commit log" name:"base" short:"B"`
	JSON        bool   `help:"Print the hashes as a JSON array" name:"json"`
}

func (c *CompleteCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	cmd := completion.NewCmd(project.NewClient(git.NewClient(ctx), ctx.Output()), os.Stdout)
	return cmd.Complete(cfg, completion.Flags{
		ProjectFile: c.ProjectFile,
		Items:       c.Items,
		Base:        c.Base,
		JSON:        c.JSON,
	})
}
