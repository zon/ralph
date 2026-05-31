package cmd

import (
	orchestrationArchitecture "github.com/zon/ralph/internal/orchestration/architecture"
)

type ArchitectureCmd struct {
	Output  string `help:"Output path for architecture.yaml" default:"architecture.yaml"`
	Verbose bool   `help:"Enable verbose logging" default:"false"`
	Model   string `help:"Override the AI model from config" name:"model" optional:""`
}

func (r *ArchitectureCmd) Run() error {
	ctx := createExecutionContext()
	ctx.SetVerbose(r.Verbose)
	ctx.SetModel(r.Model)

	owner, name := ctx.RepoOwnerAndName()

	flags := orchestrationArchitecture.ArchitectureFlags{
		Output:              r.Output,
		Verbose:             r.Verbose,
		Model:               r.Model,
		IsWorkflowExecution: ctx.IsWorkflowExecution(),
		RepoOwner:           owner,
		RepoName:            name,
		BaseBranch:          ctx.BaseBranch(),
	}

	cmd := newOrchestrationArchitectureCmd(ctx)
	return cmd.Run(flags)
}
