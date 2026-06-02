package cmd

import (
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/notify"
	"github.com/zon/ralph/internal/opencode"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

func NewLocalRunner(ctx *context.Context, baseBranch string) *orchestrationRun.Runner {
	return orchestrationRun.NewRunner(
		&project.Client{},
		NewAgentClient(ctx, opencode.New()),
		git.NewClient(ctx),
		github.NewClient(ctx, baseBranch, &github.GH{}, opencode.New()),
		&services.Client{},
		notify.NewClient(ctx),
		&SystemEnvClient{},
	)
}
