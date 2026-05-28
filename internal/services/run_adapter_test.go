package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/services"
)

func TestServicesRunAdapterRunBeforeCommands(t *testing.T) {
	adapter := &services.RunAdapter{}

	t.Run("calls RunBefore when cfg.Before is non-empty", func(t *testing.T) {
		cfg := &config.RalphConfig{
			Before: []config.Before{
				{Name: "echo", Command: "echo", Args: []string{"hello"}},
			},
		}
		err := adapter.RunBeforeCommands(cfg)
		require.NoError(t, err)
	})

	t.Run("returns nil when cfg.Before is empty", func(t *testing.T) {
		cfg := &config.RalphConfig{}
		err := adapter.RunBeforeCommands(cfg)
		require.NoError(t, err)
	})

	t.Run("returns error when a non-optional before command fails", func(t *testing.T) {
		cfg := &config.RalphConfig{
			Before: []config.Before{
				{Name: "fail", Command: "nonexistent-command-xyz"},
			},
		}
		err := adapter.RunBeforeCommands(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run before commands")
	})

	t.Run("does not fail on optional failing command", func(t *testing.T) {
		cfg := &config.RalphConfig{
			Before: []config.Before{
				{Name: "fail-optional", Command: "false", Optional: true},
			},
		}
		err := adapter.RunBeforeCommands(cfg)
		require.NoError(t, err)
	})
}

func TestServicesRunAdapterImplementsInterface(t *testing.T) {
	var _ orchestrationRun.ServicesClient = &services.RunAdapter{}
}
