package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
)

func TestNotifyClientNew(t *testing.T) {
	ctx := context.NewContext()
	client := NewClient(ctx)
	require.NotNil(t, client)
	assert.False(t, client.shouldNotify)
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

func TestNotifyClientError_WithNotificationsEnabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(false)
	ctx.SetLocal(true)
	client := NewClient(ctx)

	assert.NotPanics(t, func() {
		client.Error("test-slug")
	})
}

func TestNotifyClientSuccess_WithNotificationsEnabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(false)
	ctx.SetLocal(true)
	client := NewClient(ctx)

	assert.NotPanics(t, func() {
		client.Success("test-slug")
	})
}
