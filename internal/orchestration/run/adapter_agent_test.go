package run

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
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

	initGitRepo(t, workDir)
	makeInitialCommit(t, workDir)
	createRalphConfig(t, workDir)

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
	// Verify it implements the AgentClient interface via compilation check
	var _ AgentClient = adapter
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, cmd := range []struct {
		name string
		args []string
	}{
		{"init", []string{"init", "-b", "main"}},
		{"config", []string{"config", "user.email", "test@test.com"}},
		{"config", []string{"config", "user.name", "Test"}},
	} {
		c := exec.Command("git", cmd.args...)
		c.Dir = dir
		require.NoError(t, c.Run(), "git %v should succeed", cmd.args)
	}
}

func makeInitialCommit(t *testing.T, dir string) {
	t.Helper()
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# test"), 0644))
	c := exec.Command("git", "add", "README.md")
	c.Dir = dir
	require.NoError(t, c.Run())
	c = exec.Command("git", "commit", "-m", "initial commit")
	c.Dir = dir
	require.NoError(t, c.Run())
}

func createRalphConfig(t *testing.T, dir string) {
	t.Helper()
	ralphDir := filepath.Join(dir, ".ralph")
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configYAML := `defaultBranch: main
model: deepseek/deepseek-chat
`
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configYAML), 0644))
}
