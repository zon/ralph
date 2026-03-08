package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			if tt.wantPanic {
				assert.Panics(t, func() { Success(tt.projectName, tt.enabled) }, "Success should panic")
			} else {
				assert.NotPanics(t, func() { Success(tt.projectName, tt.enabled) }, "Success should not panic")
			}
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
			if tt.wantPanic {
				assert.Panics(t, func() { Error(tt.projectName, tt.enabled) }, "Error should panic")
			} else {
				assert.NotPanics(t, func() { Error(tt.projectName, tt.enabled) }, "Error should not panic")
			}
		})
	}
}

func TestNotificationsDisabled(t *testing.T) {
	assert.NotPanics(t, func() {
		Success("test-project", false)
		Error("test-project", false)
	}, "Should not panic when notifications are disabled")
}
