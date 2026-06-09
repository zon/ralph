package notify

import (
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/output"
)

type Client struct {
	out          *output.Client
	shouldNotify bool
	notifier     Notifier
}

func NewClient(ctx *context.Context) *Client {
	return &Client{
		out:          ctx.Output(),
		shouldNotify: ctx.ShouldNotify(),
		notifier:     &realNotifier{},
	}
}

func (a *Client) Error(slug string) {
	if !a.shouldNotify {
		return
	}

	title := "Ralph Failed"
	message := "Ralph failed for " + slug

	if err := a.notifier.Notify(title, message, "dialog-error"); err != nil {
		a.out.Warnf("Failed to send desktop notification: %v", err)
	}
}

func (a *Client) Success(slug string) {
	if !a.shouldNotify {
		return
	}

	title := "Ralph Success"
	message := "Ralph completed successfully for " + slug

	if err := a.notifier.Notify(title, message, "dialog-information"); err != nil {
		a.out.Warnf("Failed to send desktop notification: %v", err)
	}
}
