package notify

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/output"
)

func TestNotifyClientNew(t *testing.T) {
	ctx := context.NewContext()
	client := NewClient(ctx)
	require.NotNil(t, client)
	assert.False(t, client.shouldNotify)
	assert.NotNil(t, client.notifier)
}

func TestNotifyClientShouldNotifyFromContext(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(false)
	ctx.SetLocal(true)
	client := NewClient(ctx)
	assert.True(t, client.shouldNotify)
}

func TestNotifyClientImplementsInterface(t *testing.T) {
	var _ orchestrationRun.NotifyClient = &Client{}
}

func TestNotifyClientError_WithNotificationsDisabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(true)
	client := NewClient(ctx)

	assert.NotPanics(t, func() {
		client.Error("test-slug")
	})
}

func TestNotifyClientSuccess_WithNotificationsDisabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(true)
	client := NewClient(ctx)

	assert.NotPanics(t, func() {
		client.Success("test-slug")
	})
}

func TestClientSuccess_SendsCorrectNotification(t *testing.T) {
	var capturedTitle, capturedMessage, capturedIcon string
	mockNotifier := &MockNotifier{
		NotifyFn: func(title, message, appIcon string) error {
			capturedTitle, capturedMessage, capturedIcon = title, message, appIcon
			return nil
		},
	}

	client := &Client{
		out:          output.NewClient(io.Discard, io.Discard, false),
		shouldNotify: true,
		notifier:     mockNotifier,
	}

	client.Success("test-slug")
	assert.Equal(t, "Ralph Success", capturedTitle)
	assert.Equal(t, "Ralph completed successfully for test-slug", capturedMessage)
	assert.Equal(t, "dialog-information", capturedIcon)
}

func TestClientError_SendsCorrectNotification(t *testing.T) {
	var capturedTitle, capturedMessage, capturedIcon string
	mockNotifier := &MockNotifier{
		NotifyFn: func(title, message, appIcon string) error {
			capturedTitle, capturedMessage, capturedIcon = title, message, appIcon
			return nil
		},
	}

	client := &Client{
		out:          output.NewClient(io.Discard, io.Discard, false),
		shouldNotify: true,
		notifier:     mockNotifier,
	}

	client.Error("test-slug")
	assert.Equal(t, "Ralph Failed", capturedTitle)
	assert.Equal(t, "Ralph failed for test-slug", capturedMessage)
	assert.Equal(t, "dialog-error", capturedIcon)
}

func TestClientSuccess_WhenNotifyFails_CallsWarnf(t *testing.T) {
	mockErr := errors.New("notify error")
	mockNotifier := &MockNotifier{
		NotifyFn: func(_, _, _ string) error {
			return mockErr
		},
	}

	out := output.NewClient(io.Discard, io.Discard, false)
	client := &Client{
		out:          out,
		shouldNotify: true,
		notifier:     mockNotifier,
	}

	assert.NotPanics(t, func() {
		client.Success("test-slug")
	})
}

func TestClientError_WhenNotifyFails_CallsWarnf(t *testing.T) {
	mockErr := errors.New("notify error")
	mockNotifier := &MockNotifier{
		NotifyFn: func(_, _, _ string) error {
			return mockErr
		},
	}

	out := output.NewClient(io.Discard, io.Discard, false)
	client := &Client{
		out:          out,
		shouldNotify: true,
		notifier:     mockNotifier,
	}

	assert.NotPanics(t, func() {
		client.Error("test-slug")
	})
}

func TestClientSuccess_WhenNotificationsDisabled_DoesNotCallNotify(t *testing.T) {
	callCount := 0
	mockNotifier := &MockNotifier{
		NotifyFn: func(_, _, _ string) error {
			callCount++
			return nil
		},
	}

	client := &Client{
		out:          output.NewClient(io.Discard, io.Discard, false),
		shouldNotify: false,
		notifier:     mockNotifier,
	}

	client.Success("test-slug")
	assert.Equal(t, 0, callCount)
}

func TestClientError_WhenNotificationsDisabled_DoesNotCallNotify(t *testing.T) {
	callCount := 0
	mockNotifier := &MockNotifier{
		NotifyFn: func(_, _, _ string) error {
			callCount++
			return nil
		},
	}

	client := &Client{
		out:          output.NewClient(io.Discard, io.Discard, false),
		shouldNotify: false,
		notifier:     mockNotifier,
	}

	client.Error("test-slug")
	assert.Equal(t, 0, callCount)
}

