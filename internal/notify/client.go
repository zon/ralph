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
		shouldNotify: (ctx.IsLocal() || ctx.ShouldFollow()) && !ctx.NoNotify(),
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
		a.warnNotifyFailure(err)
	}
}

// warnNotifyFailure reports a failed notification without letting the best-effort
// error path panic. A context without an output client leaves out nil.
func (a *Client) warnNotifyFailure(err error) {
	if a.out == nil {
		return
	}
	a.out.Warnf("Failed to send desktop notification: %v", err)
}

func (a *Client) Success(slug string) {
	if !a.shouldNotify {
		return
	}

	title := "Ralph Success"
	message := "Ralph completed successfully for " + slug

	if err := a.notifier.Notify(title, message, "dialog-information"); err != nil {
		a.warnNotifyFailure(err)
	}
}
