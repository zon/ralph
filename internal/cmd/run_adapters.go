package cmd

import (
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/run"
	"github.com/zon/ralph/internal/workspace"
)

func newOrchestrationRunCmd(ctx *execcontext.Context) *orchestrationRun.RunCmd {
	return orchestrationRun.NewRunCmd(
		&workspace.Client{},
		&project.Client{},
		git.NewClient(ctx),
		&config.Client{},
		run.NewLocalRunnerClient(ctx),
		run.NewRemoteRunnerClient(ctx),
	)
}
