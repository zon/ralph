package cmd

import (
	"os"

	"github.com/zon/ralph/internal/opencode"
	orchvalidate "github.com/zon/ralph/internal/orchestration/validate"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/validate"
)

type ValidateCmd struct {
	ProjectFile string `arg:"" help:"Path to project YAML file"`
}

func (v *ValidateCmd) Run() error {
	orchestrator := newValidateOrchestrator()
	proj, err := orchestrator.Run(v.ProjectFile)
	if err != nil {
		return err
	}

	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	ctx.Output().Successf("Project '%s' is valid (%d requirements)", proj.Slug, len(proj.Requirements))
	return nil
}

type validateValidatorAdapter struct {
	validator *validate.Validator
}

func (a *validateValidatorAdapter) Validate(path string) (*project.Project, error) {
	return a.validator.Validate(path)
}

func newValidateOrchestrator() *orchvalidate.ValidateCmd {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	return orchvalidate.New(
		&validateValidatorAdapter{
			validator: validate.New(ctx, opencode.New()),
		},
	)
}
