package cmd

import (
	"fmt"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/output"
	orchestrationCommand "github.com/zon/ralph/internal/orchestration/command"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

func ExecuteCommand(ctx *context.Context, cleanupRegistrar func(func()), setup *CommandSetup) error {
	if !ctx.IsLocal() {
		return orchestrationCommand.ExecuteRemoteCommand(ctx, argo.NewClient())
	}

	if err := infrastructureRunBeforeCommands(ctx.Output(), setup.Config); err != nil {
		return err
	}

	if err := runCommand(setup.Command); err != nil {
		notify.NewClient(ctx).Error("command")
		return err
	}

	notify.NewClient(ctx).Success("command")
	return nil
}

func Execute(ctx *context.Context, cleanupRegistrar func(func()), setup *ExecutionSetup) error {
	if !ctx.IsLocal() {
		return NewRemoteRunner(ctx).Run(project.ForProjectInput(setup.Project), orchestrationRun.RunRemoteFlags{Follow: ctx.ShouldFollow()})
	}

	return NewLocalRunner(ctx, setup.BaseBranch).RunLocal(project.ForProjectInput(setup.Project), setup.Config)
}

func infrastructureRunBeforeCommands(out *output.Client, cfg *config.RalphConfig) error {
	if len(cfg.Before) > 0 {
		if err := services.RunBefore(out, cfg.Before); err != nil {
			return fmt.Errorf("failed to run before commands: %w", err)
		}
	}
	return nil
}
