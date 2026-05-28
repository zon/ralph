package run

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
)

func TestNotifyClientAdapterNewAdapter(t *testing.T) {
	ctx := context.NewContext()
	adapter := NewNotifyClientAdapter(ctx)
	require.NotNil(t, adapter)
	assert.False(t, adapter.shouldNotify)
}

func TestNotifyClientAdapterShouldNotifyFromContext(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(false)
	ctx.SetLocal(true)
	adapter := NewNotifyClientAdapter(ctx)
	assert.True(t, adapter.shouldNotify)
}

func TestNotifyClientAdapterImplementsInterface(t *testing.T) {
	var _ NotifyClient = &NotifyClientAdapter{}
}

func TestNotifyClientAdapterError_WithNotificationsDisabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(true)
	adapter := NewNotifyClientAdapter(ctx)

	assert.NotPanics(t, func() {
		adapter.Error("test-slug")
	})
}

func TestNotifyClientAdapterSuccess_WithNotificationsDisabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(true)
	adapter := NewNotifyClientAdapter(ctx)

	assert.NotPanics(t, func() {
		adapter.Success("test-slug")
	})
}

func TestNotifyClientAdapterError_WithNotificationsEnabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(false)
	ctx.SetLocal(true)
	adapter := NewNotifyClientAdapter(ctx)

	assert.NotPanics(t, func() {
		adapter.Error("test-slug")
	})
}

func TestNotifyClientAdapterSuccess_WithNotificationsEnabled(t *testing.T) {
	ctx := context.NewContext()
	ctx.SetNoNotify(false)
	ctx.SetLocal(true)
	adapter := NewNotifyClientAdapter(ctx)

	assert.NotPanics(t, func() {
		adapter.Success("test-slug")
	})
}

func TestNewRunnerWiresAllAdapters(t *testing.T) {
	ctx := context.NewContext()
	baseBranch := "main"
	runner := NewRunner(ctx, baseBranch)

	require.NotNil(t, runner)
	require.NotNil(t, runner.project)
	require.NotNil(t, runner.ai)
	require.NotNil(t, runner.git)
	require.NotNil(t, runner.github)
	require.NotNil(t, runner.services)
	require.NotNil(t, runner.notify)

	_, ok := runner.project.(*ProjectClientAdapter)
	assert.True(t, ok, "project should be *ProjectClientAdapter")

	_, ok = runner.ai.(*AgentClientAdapter)
	assert.True(t, ok, "ai should be *AgentClientAdapter")

	_, ok = runner.git.(*GitClientAdapter)
	assert.True(t, ok, "git should be *GitClientAdapter")

	_, ok = runner.github.(*GitHubClientAdapter)
	assert.True(t, ok, "github should be *GitHubClientAdapter")

	_, ok = runner.services.(*ServicesClientAdapter)
	assert.True(t, ok, "services should be *ServicesClientAdapter")

	_, ok = runner.notify.(*NotifyClientAdapter)
	assert.True(t, ok, "notify should be *NotifyClientAdapter")
}
