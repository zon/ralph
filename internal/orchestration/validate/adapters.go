package validate

import (
	"os"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/projectfile"
)

type projectFile struct{}

func (projectFile) Parse(path string) (*projectfile.Document, error) {
	return projectfile.Parse(path)
}

func (projectFile) ResolveItems(doc *projectfile.Document, query string) ([]any, error) {
	return projectfile.ResolveItems(doc, query)
}

func (projectFile) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (projectFile) WriteCanonical(path string, doc *projectfile.Document) error {
	return projectfile.WriteCanonical(path, doc)
}

func (projectFile) Remove(path string) error {
	return projectfile.Remove(path)
}

func (projectFile) CanonicalPath(path string) string {
	return projectfile.CanonicalPath(path)
}

type agentClient struct {
	ctx *context.Context
	oc  opencode.OCClient
}

func (a *agentClient) FixProject(path string, parseErr error, model string) error {
	prompt, err := ai.BuildProjectFixPrompt(path, parseErr)
	if err != nil {
		return err
	}
	return ai.RunAgentWithModel(a.ctx, a.oc, prompt, model)
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

func New(ctx *context.Context, oc opencode.OCClient) *Validator {
	return &Validator{
		file:     &projectFile{},
		agent:    &agentClient{ctx: ctx, oc: oc},
		model:    resolveConfigModel(),
		reporter: ctx.Output(),
	}
}
