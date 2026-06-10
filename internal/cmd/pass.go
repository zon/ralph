package cmd

import (
	"github.com/zon/ralph/internal/orchestration/pass"
)

type PassCmd struct {
	ProjectFile string `arg:"" help:"Path to project YAML file"`
	Slug        string `arg:"" help:"Requirement slug"`
	False       bool   `name:"false" help:"Mark as failing instead of passing"`
}

func (c *PassCmd) Run() error {
	p := &pass.PassCmd{
		ProjectFile: c.ProjectFile,
		Slug:        c.Slug,
		False:       c.False,
	}
	_, err := p.Run()
	return err
}
