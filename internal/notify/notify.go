package notify

import (
	"github.com/gen2brain/beeep"
	"github.com/zon/ralph/internal/output"
)

func init() {
	// Set the application name for notifications
	// This appears in the notification header on Ubuntu and other Linux systems
	beeep.AppName = "Ralph"
}

// Notifier wraps beeep notification functionality behind an
// interface so it can be substituted in tests.
type Notifier interface {
	Notify(title, message, appIcon string) error
}

type realNotifier struct{}

func (r *realNotifier) Notify(title, message, appIcon string) error {
	return beeep.Notify(title, message, appIcon)
}

var _ Notifier = (*realNotifier)(nil)

var defaultNotifier Notifier = &realNotifier{}

// Success sends a success notification
// If notify is disabled, this is a no-op
func Success(out *output.Client, projectName string, enabled bool) {
	if !enabled {
		return
	}

	title := "Ralph Success"
	message := "Ralph completed successfully for " + projectName

	if err := defaultNotifier.Notify(title, message, "dialog-information"); err != nil {
		out.Warnf("Failed to send desktop notification: %v", err)
	}
}

// Error sends an error notification
// If notify is disabled, this is a no-op
func Error(out *output.Client, projectName string, enabled bool) {
	if !enabled {
		return
	}

	title := "Ralph Failed"
	message := "Ralph failed for " + projectName

	if err := defaultNotifier.Notify(title, message, "dialog-error"); err != nil {
		out.Warnf("Failed to send desktop notification: %v", err)
	}
}
