package pass

import (
	"github.com/zon/ralph/internal/project"
)

type PassCmd struct {
	ProjectFile string `arg:""`
	Slug        string `arg:""`
	False       bool   `name:"false"`
}

func (c *PassCmd) Run() (*project.Project, error) {
	proj, err := project.LoadProject(c.ProjectFile)
	if err != nil {
		return nil, err
	}

	if err := project.UpdateRequirementStatus(proj, c.Slug, !c.False); err != nil {
		return nil, err
	}

	if err := project.SaveProject(c.ProjectFile, proj); err != nil {
		return nil, err
	}

	return proj, nil
}
