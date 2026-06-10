package cmd

import (
	"os"

	"github.com/zon/ralph/internal/orchestration/pass"
	"github.com/zon/ralph/internal/output"
)

type PassCmd struct {
	ProjectFile string `arg:"" help:"Path to project YAML file"`
	Slug        string `arg:"" help:"Requirement slug"`
	False       bool   `name:"false" help:"Mark as failing instead of passing"`
}

func (c *PassCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))

	p := &pass.PassCmd{
		ProjectFile: c.ProjectFile,
		Slug:        c.Slug,
		False:       c.False,
	}
	proj, err := p.Run()
	if err != nil {
		return err
	}

	status := "failing"
	for _, req := range proj.Requirements {
		if req.Slug == c.Slug && req.Passing {
			status = "passing"
			break
		}
	}

	ctx.Output().Successf("Requirement '%s' is %s", c.Slug, status)
	return nil
}
