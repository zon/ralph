package cmd

import (
	"fmt"

	"github.com/zon/ralph/internal/ai"
	"github.com/zon/ralph/internal/logger"
)

type ArchitectureCmd struct {
	Output  string `help:"Output path for architecture.yaml" default:"architecture.yaml"`
	Verbose bool   `help:"Enable verbose logging" default:"false"`
	Model   string `help:"Override the AI model from config" name:"model" optional:""`
}

func (r *ArchitectureCmd) Run() error {
	if r.Verbose {
		logger.SetVerbose(true)
	}

	ctx := createExecutionContext()
	ctx.SetVerbose(r.Verbose)
	ctx.SetModel(r.Model)

	prompt, err := ai.BuildArchitecturePrompt(r.Output)
	if err != nil {
		return fmt.Errorf("failed to build architecture prompt: %w", err)
	}

	if err := ai.RunAgent(ctx, prompt); err != nil {
		return fmt.Errorf("architecture generation failed: %w", err)
	}

	logger.Successf("Architecture written to %s", r.Output)
	return nil
}
