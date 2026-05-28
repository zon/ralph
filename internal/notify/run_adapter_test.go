package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
)

func TestNotifyRunAdapterNewAdapter(t *testing.T) {
	ctx := context.NewContext()
	adapter := NewRunAdapter(ctx)
	require.NotNil(t, adapter)
	assert.False(t, adapter.shouldNotify)
}

func TestNotifyRunAdapterShouldNotifyFromContext(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(false)
	ctx.SetLocal(true)
	adapter := NewRunAdapter(ctx)
	assert.True(t, adapter.shouldNotify)
}

func TestNotifyRunAdapterImplementsInterface(t *testing.T) {
	var _ orchestrationRun.NotifyClient = &RunAdapter{}
}

func TestNotifyRunAdapterError_WithNotificationsDisabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(true)
	adapter := NewRunAdapter(ctx)

	assert.NotPanics(t, func() {
		adapter.Error("test-slug")
	})
}

func TestNotifyRunAdapterSuccess_WithNotificationsDisabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(true)
	adapter := NewRunAdapter(ctx)

	assert.NotPanics(t, func() {
		adapter.Success("test-slug")
	})
}

func TestNotifyRunAdapterError_WithNotificationsEnabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(false)
	ctx.SetLocal(true)
	adapter := NewRunAdapter(ctx)

	assert.NotPanics(t, func() {
		adapter.Error("test-slug")
	})
}

func TestNotifyRunAdapterSuccess_WithNotificationsEnabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(false)
	ctx.SetLocal(true)
	adapter := NewRunAdapter(ctx)

	assert.NotPanics(t, func() {
		adapter.Success("test-slug")
	})
}
