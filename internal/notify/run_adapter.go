package notify

import (
	"github.com/zon/ralph/internal/context"
)

type RunAdapter struct {
	shouldNotify bool
}

func NewRunAdapter(ctx *context.Context) *RunAdapter {
	return &RunAdapter{
		shouldNotify: ctx.ShouldNotify(),
	}
}

func (a *RunAdapter) Error(slug string) {
	Error(slug, a.shouldNotify)
}

func (a *RunAdapter) Success(slug string) {
	Success(slug, a.shouldNotify)
}
