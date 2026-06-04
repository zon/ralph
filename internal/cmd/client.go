package cmd

import (
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
)

type LocalRunnerClient struct {
	ctx *execcontext.Context
}

func NewLocalRunnerClient(ctx *execcontext.Context) *LocalRunnerClient {
	return &LocalRunnerClient{ctx: ctx}
}

func (c *LocalRunnerClient) RunLocal(proj *project.Project, cfg *config.RalphConfig) error {
	runner := NewLocalRunner(c.ctx, proj.BaseBranch)
	return runner.RunLocal(proj, cfg)
}

type RemoteRunnerClient struct {
	ctx *execcontext.Context
}

func NewRemoteRunnerClient(ctx *execcontext.Context) *RemoteRunnerClient {
	return &RemoteRunnerClient{ctx: ctx}
}

func (c *RemoteRunnerClient) Run(proj *project.Project, flags orchestrationRun.RunRemoteFlags) error {
	runner := NewRemoteRunner(c.ctx)
	return runner.Run(proj, flags)
}
