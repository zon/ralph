package notify

import (
	"github.com/gen2brain/beeep"
	"github.com/zon/ralph/internal/logger"
)

func init() {
	// Set the application name for notifications
	// This appears in the notification header on Ubuntu and other Linux systems
	beeep.AppName = "Ralph"
}

// Success sends a success notification
// If notify is disabled, this is a no-op
func Success(projectName string, enabled bool) {
	if !enabled {
		return
	}

	title := "Ralph Success"
	message := "Ralph completed successfully for " + projectName

	// Use dialog-information icon (stock icon available on most Linux systems)
	if err := beeep.Notify(title, message, "dialog-information"); err != nil {
		// Gracefully handle notification failures
		logger.Warningf("Failed to send desktop notification: %v", err)
	}
}

// Error sends an error notification
// If notify is disabled, this is a no-op
func Error(projectName string, enabled bool) {
	if !enabled {
		return
	}

	title := "Ralph Failed"
	message := "Ralph failed for " + projectName

	// Use dialog-error icon (stock icon available on most Linux systems)
	if err := beeep.Notify(title, message, "dialog-error"); err != nil {
		// Gracefully handle notification failures
		logger.Warningf("Failed to send desktop notification: %v", err)
	}
}
