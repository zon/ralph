package notify

import (
	"testing"
)

func TestSuccess(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		enabled     bool
		wantPanic   bool
	}{
		{
			name:        "notifications disabled",
			projectName: "test-project",
			enabled:     false,
			wantPanic:   false,
		},
		{
			name:        "empty project name notifications disabled",
			projectName: "",
			enabled:     false,
			wantPanic:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantPanic {
						t.Errorf("Success() panicked unexpectedly: %v", r)
					}
				}
			}()

			// Should not panic or return error
			// NOTE: Tests should never use enabled=true to avoid real desktop notifications
			Success(tt.projectName, tt.enabled)
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		enabled     bool
		wantPanic   bool
	}{
		{
			name:        "notifications disabled",
			projectName: "test-project",
			enabled:     false,
			wantPanic:   false,
		},
		{
			name:        "empty project name notifications disabled",
			projectName: "",
			enabled:     false,
			wantPanic:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantPanic {
						t.Errorf("Error() panicked unexpectedly: %v", r)
					}
				}
			}()

			// Should not panic or return error
			// NOTE: Tests should never use enabled=true to avoid real desktop notifications
			Error(tt.projectName, tt.enabled)
		})
	}
}

func TestNotificationsDisabled(t *testing.T) {
	// Test that when notifications are disabled, no error occurs
	// This is important for graceful degradation
	Success("test-project", false)
	Error("test-project", false)

	// If we got here without panicking, the test passed
}
