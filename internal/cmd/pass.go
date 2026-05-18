package cmd

import (
	"github.com/zon/ralph/internal/project"
)

type PassCmd struct {
	ProjectFile string `arg:""`
	Slug        string `arg:""`
	False       bool   `name:"false"`
}

func (c *PassCmd) Run() error {
	proj, err := project.LoadProject(c.ProjectFile)
	if err != nil {
		return err
	}

	if err := project.UpdateRequirementStatus(proj, c.Slug, !c.False); err != nil {
		return err
	}

	return project.SaveProject(c.ProjectFile, proj)
}