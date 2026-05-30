package notify

import (
	"github.com/zon/ralph/internal/context"
)

type Client struct {
	shouldNotify bool
}

func NewClient(ctx *context.Context) *Client {
	return &Client{
		shouldNotify: ctx.ShouldNotify(),
	}
}

func (a *Client) Error(slug string) {
	Error(slug, a.shouldNotify)
}

func (a *Client) Success(slug string) {
	Success(slug, a.shouldNotify)
}
