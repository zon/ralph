package cmd

import (
	"github.com/zon/ralph/internal/orchestration/pass"
	"github.com/zon/ralph/internal/project"
)

type PassCmd struct {
	ProjectFile string `arg:""`
	Slug        string `arg:""`
	False       bool   `name:"false"`
}

func (c *PassCmd) Run() error {
	orchestrator := newPassOrchestrator()
	return orchestrator.Run(c.ProjectFile, c.Slug, !c.False)
}

type passProjectLoaderAdapter struct{}

func (a *passProjectLoaderAdapter) LoadProject(path string) (*project.Project, error) {
	return project.LoadProject(path)
}

type passProjectSaverAdapter struct{}

func (a *passProjectSaverAdapter) SaveProject(path string, p *project.Project) error {
	return project.SaveProject(path, p)
}

func newPassOrchestrator() *pass.PassCmd {
	return pass.New(
		&passProjectLoaderAdapter{},
		&passProjectSaverAdapter{},
	)
}