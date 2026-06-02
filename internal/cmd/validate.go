package cmd

import (
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/validate"
)

type ValidateCmd struct {
	ProjectFile string `arg:"" help:"Path to project YAML file"`
}

func (v *ValidateCmd) Run() error {
	ctx := createExecutionContext()
	validator := validate.New(ctx, opencode.New())
	proj, err := validator.Validate(v.ProjectFile)
	if err != nil {
		return err
	}

	logger.Successf("Project '%s' is valid (%d requirements)", proj.Slug, len(proj.Requirements))
	return nil
}
