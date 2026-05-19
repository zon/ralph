package validate

import (
	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
)

type projectClient struct{}

func (projectClient) Load(path string) (*project.Project, error) {
	return project.LoadProject(path)
}

func (projectClient) Save(path string, proj *project.Project) error {
	return project.SaveProject(path, proj)
}

type agentClient struct {
	ctx *context.Context
}

func (a *agentClient) FixProject(path string, loadErr error) error {
	prompt, err := ai.BuildProjectFixPrompt(path, loadErr)
	if err != nil {
		return err
	}
	return ai.RunAgent(a.ctx, prompt)
}

func New(ctx *context.Context) *Validator {
	return &Validator{
		project: &projectClient{},
		agent:   &agentClient{ctx: ctx},
	}
}