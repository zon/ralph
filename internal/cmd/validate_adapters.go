package cmd

import (
	"os"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	orchestrationValidate "github.com/zon/ralph/internal/orchestration/validate"
	"github.com/zon/ralph/internal/projectfile"
)

func newValidateValidator(ctx *execcontext.Context, oc opencode.OCClient) *orchestrationValidate.Validator {
	return orchestrationValidate.NewValidator(
		&projectFileAdapter{},
		&validateAgentClient{ctx: ctx, oc: oc},
		resolveValidateModel(),
		ctx.Output(),
	)
}

type projectFileAdapter struct{}

func (projectFileAdapter) Parse(path string) (*projectfile.Document, error) {
	return projectfile.Parse(path)
}

func (projectFileAdapter) ResolveItems(doc *projectfile.Document, query string) ([]any, error) {
	return projectfile.ResolveItems(doc, query)
}

func (projectFileAdapter) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (projectFileAdapter) WriteCanonical(path string, doc *projectfile.Document) error {
	return projectfile.WriteCanonical(path, doc)
}

func (projectFileAdapter) Remove(path string) error {
	return projectfile.Remove(path)
}

func (projectFileAdapter) CanonicalPath(path string) string {
	return projectfile.CanonicalPath(path)
}

type validateAgentClient struct {
	ctx *execcontext.Context
	oc  opencode.OCClient
}

func (a *validateAgentClient) FixProject(path string, parseErr error, model string) error {
	prompt, err := ai.BuildProjectFixPrompt(path, parseErr)
	if err != nil {
		return err
	}
	return ai.RunAgentWithModel(a.ctx, a.oc, prompt, model)
}

// resolveValidateModel returns the model the validate repair prompt runs with:
// the validate-specific model when configured, otherwise the top-level model.
func resolveValidateModel() string {
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return ""
	}
	if ralphConfig.Validate.Model != "" {
		return ralphConfig.Validate.Model
	}
	return ralphConfig.Model
}
