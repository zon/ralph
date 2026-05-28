package run

import (
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/notify"
)

// NotifyClientAdapter adapts notify functions to the NotifyClient interface.
type NotifyClientAdapter struct {
	shouldNotify bool
}

func NewNotifyClientAdapter(ctx *context.Context) *NotifyClientAdapter {
	return &NotifyClientAdapter{
		shouldNotify: ctx.ShouldNotify(),
	}
}

func (a *NotifyClientAdapter) Error(slug string) {
	notify.Error(slug, a.shouldNotify)
}

func (a *NotifyClientAdapter) Success(slug string) {
	notify.Success(slug, a.shouldNotify)
}

func NewRunner(ctx *context.Context, baseBranch string) *Runner {
	return &Runner{
		project:  &ProjectClientAdapter{},
		ai:       NewAgentClientAdapter(ctx),
		git:      NewGitClientAdapter(ctx),
		github:   NewGitHubClientAdapter(ctx, baseBranch),
		services: &ServicesClientAdapter{},
		notify:   NewNotifyClientAdapter(ctx),
	}
}
