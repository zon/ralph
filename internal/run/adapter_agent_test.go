package run

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func TestAgentClientAdapterIsFatal(t *testing.T) {
	ctx := context.NewContext()
	adapter := NewAgentClientAdapter(ctx)

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, adapter.IsFatal(nil))
	})

	t.Run("detects Insufficient Balance", func(t *testing.T) {
		err := errors.New("opencode execution failed: Insufficient Balance")
		assert.True(t, adapter.IsFatal(err))
	})

	t.Run("detects lowercase insufficient balance", func(t *testing.T) {
		err := errors.New("opencode execution failed: insufficient balance")
		assert.True(t, adapter.IsFatal(err))
	})

	t.Run("detects billing error", func(t *testing.T) {
		err := errors.New("opencode execution failed: billing error")
		assert.True(t, adapter.IsFatal(err))
	})

	t.Run("detects account error", func(t *testing.T) {
		err := errors.New("opencode execution failed: account error")
		assert.True(t, adapter.IsFatal(err))
	})

	t.Run("detects payment required", func(t *testing.T) {
		err := errors.New("opencode execution failed: payment required")
		assert.True(t, adapter.IsFatal(err))
	})

	t.Run("detects quota exceeded", func(t *testing.T) {
		err := errors.New("opencode execution failed: quota exceeded")
		assert.True(t, adapter.IsFatal(err))
	})

	t.Run("returns false for regular error", func(t *testing.T) {
		err := errors.New("some other error")
		assert.False(t, adapter.IsFatal(err))
	})
}

func TestAgentClientAdapterIterate_MockAI(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := t.TempDir()
	t.Chdir(workDir)

	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	testutil.CreateRalphConfig(t, workDir)

	projectYAML := `slug: test-project
title: Test project
requirements:
  - slug: req-1
    description: Test requirement
    items:
      - Item 1
    passing: false
`
	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := context.NewContext()
	ctx.SetProjectFile("test-project.yaml")
	adapter := NewAgentClientAdapter(ctx)

	proj := &project.Project{Slug: "test-project", MaxIterations: 1}
	err := adapter.Iterate(proj)
	require.NoError(t, err)
}

func TestAgentClientAdapterNewAdapter(t *testing.T) {
	ctx := context.NewContext()
	adapter := NewAgentClientAdapter(ctx)
	require.NotNil(t, adapter)
	var _ orchestrationRun.AgentClient = adapter
}
