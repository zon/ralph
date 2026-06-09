package notify

import (
	"github.com/gen2brain/beeep"
)

func init() {
	beeep.AppName = "Ralph"
}

type Notifier interface {
	Notify(title, message, appIcon string) error
}

type realNotifier struct{}

func (r *realNotifier) Notify(title, message, appIcon string) error {
	return beeep.Notify(title, message, appIcon)
}

var _ Notifier = (*realNotifier)(nil)

var defaultNotifier Notifier = &realNotifier{}
