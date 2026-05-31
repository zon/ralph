package validate

import (
	"context"
	"os"
	"strings"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/eino"
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
	ctx *execcontext.Context
}

func (a *agentClient) FixProject(path string, loadErr error, model string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	prompt, err := ai.BuildProjectFixPrompt(path, content, loadErr)
	if err != nil {
		return err
	}
	response, err := eino.Complete(context.Background(), model, prompt)
	if err != nil {
		return err
	}
	yaml := extractYAML(response)
	return os.WriteFile(path, yaml, 0644)
}

func extractYAML(response string) []byte {
	trimmed := strings.TrimSpace(response)

	if strings.HasPrefix(trimmed, "```") && strings.HasSuffix(trimmed, "```") && len(trimmed) > 6 {
		rest := trimmed[3:]
		if idx := strings.IndexByte(rest, '\n'); idx != -1 {
			rest = rest[idx+1:]
		} else {
			rest = ""
		}
		if len(rest) >= 3 {
			rest = rest[:len(rest)-3]
		}
		return []byte(strings.TrimSpace(rest))
	}

	return []byte(trimmed)
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

func New(ctx *execcontext.Context) *Validator {
	return &Validator{
		project: &projectClient{},
		agent:   &agentClient{ctx: ctx},
		model:   resolveConfigModel(),
	}
}