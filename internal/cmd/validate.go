package cmd

import (
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/output"
)

type ValidateCmd struct {
	ProjectFile string `arg:"" help:"Path to project YAML file"`
	Items       string `help:"jq query selecting the item array from the project file (default: from config or .)" name:"items" optional:"" short:"i"`
}

func (v *ValidateCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))

	query := v.Items
	if query == "" {
		if ralphConfig, err := config.LoadConfig(); err == nil {
			query = ralphConfig.ResolveItems("")
		}
		if query == "" {
			query = "."
		}
	}

	validator := newValidateValidator(ctx, opencode.New())
	res, err := validator.Validate(v.ProjectFile, query)
	if err != nil {
		return err
	}

	ctx.Output().Successf("Project file '%s' is valid (%d items)", res.Path, res.ItemCount)
	return nil
}
