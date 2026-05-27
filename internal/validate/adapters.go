package validate

import (
	"os"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
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

func (projectClient) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

type agentClient struct {
	ctx *context.Context
}

func (a *agentClient) FixProject(path string, loadErr error, model string) error {
	prompt, err := ai.BuildProjectFixPrompt(path, loadErr)
	if err != nil {
		return err
	}
	return ai.RunAgentWithModel(a.ctx, prompt, model)
}

func resolveConfigModel() string {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return ""
	}
	if ralphConfig.Validate.Model != "" {
		return ralphConfig.Validate.Model
	}
	return ralphConfig.Model
}

func New(ctx *context.Context) *Validator {
	return &Validator{
		project: &projectClient{},
		agent:   &agentClient{ctx: ctx},
		model:   resolveConfigModel(),
	}
}