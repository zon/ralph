package run

import (
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/notify"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/services"
)

func NewLocalRunner(ctx *context.Context, baseBranch string) *orchestrationRun.Runner {
	return orchestrationRun.NewRunner(
		&project.RunAdapter{},
		NewAgentClientAdapter(ctx),
		git.NewRunAdapter(ctx),
		github.NewRunAdapter(ctx, baseBranch),
		&services.RunAdapter{},
		notify.NewRunAdapter(ctx),
	)
}
