package notify

import (
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/output"
)

type Client struct {
	out          *output.Client
	shouldNotify bool
}

func NewClient(ctx *context.Context) *Client {
	return &Client{
		out:          ctx.Output(),
		shouldNotify: ctx.ShouldNotify(),
	}
}

func (a *Client) Error(slug string) {
	Error(a.out, slug, a.shouldNotify)
}

func (a *Client) Success(slug string) {
	Success(a.out, slug, a.shouldNotify)
}
