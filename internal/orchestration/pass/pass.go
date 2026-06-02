package pass

import (
	"github.com/zon/ralph/internal/project"
)

type ProjectLoader interface {
	LoadProject(path string) (*project.Project, error)
}

type ProjectSaver interface {
	SaveProject(path string, p *project.Project) error
}

type PassCmd struct {
	loader ProjectLoader
	saver  ProjectSaver
}

func New(loader ProjectLoader, saver ProjectSaver) *PassCmd {
	return &PassCmd{loader: loader, saver: saver}
}

func (c *PassCmd) Run(projectFile, slug string, passing bool) error {
	proj, err := c.loader.LoadProject(projectFile)
	if err != nil {
		return err
	}

	if err := project.UpdateRequirementStatus(proj, slug, passing); err != nil {
		return err
	}

	return c.saver.SaveProject(projectFile, proj)
}
