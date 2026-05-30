package run

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func TestAgentClientIsFatal(t *testing.T) {
	ctx := context.NewContext()
	client := NewAgentClient(ctx)

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, client.IsFatal(nil))
	})

	t.Run("detects Insufficient Balance", func(t *testing.T) {
		err := errors.New("opencode execution failed: Insufficient Balance")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects lowercase insufficient balance", func(t *testing.T) {
		err := errors.New("opencode execution failed: insufficient balance")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects billing error", func(t *testing.T) {
		err := errors.New("opencode execution failed: billing error")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects account error", func(t *testing.T) {
		err := errors.New("opencode execution failed: account error")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects payment required", func(t *testing.T) {
		err := errors.New("opencode execution failed: payment required")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("detects quota exceeded", func(t *testing.T) {
		err := errors.New("opencode execution failed: quota exceeded")
		assert.True(t, client.IsFatal(err))
	})

	t.Run("returns false for regular error", func(t *testing.T) {
		err := errors.New("some other error")
		assert.False(t, client.IsFatal(err))
	})
}

func TestAgentClientPickAndDevelop_MockAI(t *testing.T) {
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
	client := NewAgentClient(ctx)

	proj := &project.Project{Slug: "test-project", MaxIterations: 1}
	req, err := client.RunPicker(proj)
	require.NoError(t, err)
	require.NotEmpty(t, req)

	err = client.RunDeveloper(proj, req)
	require.NoError(t, err)
}

func TestAgentClientImplementsInterface(t *testing.T) {
	ctx := context.NewContext()
	client := NewAgentClient(ctx)
	require.NotNil(t, client)
	var _ orchestrationRun.AIClient = client
}

func TestAgentClientCollectorInitialized(t *testing.T) {
	ctx := context.NewContext()
	client := NewAgentClient(ctx)

	require.NotNil(t, client.collector)
	require.Empty(t, client.collector.IDs())

	stored := opencode.SessionCollectorFrom(client.ctx.GoContext())
	require.Same(t, client.collector, stored)
}

func TestAgentClientCollectorStoredInGoContext(t *testing.T) {
	ctx := context.NewContext()
	client := NewAgentClient(ctx)

	client.collector.Append("sess-test")
	ids := client.collector.IDs()
	require.Len(t, ids, 1)
	require.Equal(t, "sess-test", ids[0])

	stored := opencode.SessionCollectorFrom(client.ctx.GoContext())
	storedIDs := stored.IDs()
	require.Len(t, storedIDs, 1)
	require.Equal(t, "sess-test", storedIDs[0])
}

func TestAgentClientPrintStatsEmpty(t *testing.T) {
	ctx := context.NewContext()
	client := NewAgentClient(ctx)

	client.PrintStats()
}

func TestAgentClientPrintStats(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")
	scriptContent := `#!/bin/bash
if [ "$1" = "export" ]; then
  echo '{"info":{"cost":0.001,"tokens":{"input":100,"output":200}}}'
  exit 0
fi
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	ctx := context.NewContext()
	client := NewAgentClient(ctx)

	client.collector.Append("sess-one")
	client.collector.Append("sess-two")

	client.PrintStats()
}

func TestAgentClientPrintStatsPartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")
	scriptContent := `#!/bin/bash
if [ "$1" = "export" ]; then
  if [ "$2" = "sess-good" ]; then
    echo '{"info":{"cost":0.002,"tokens":{"input":50,"output":75}}}'
    exit 0
  fi
  echo "error: session not found" >&2
  exit 1
fi
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)

	ctx := context.NewContext()
	client := NewAgentClient(ctx)

	client.collector.Append("sess-bad")
	client.collector.Append("sess-good")
	client.collector.Append("sess-unknown")

	client.PrintStats()
}
