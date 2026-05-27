package run

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func TestNewRunnerWiresAllAdapters(t *testing.T) {
	ctx := testutil.NewContext()
	runner := NewRunner(ctx, "main")

	require.NotNil(t, runner)
	assert.NotNil(t, runner.project)
	assert.NotNil(t, runner.ai)
	assert.NotNil(t, runner.git)
	assert.NotNil(t, runner.github)
	assert.NotNil(t, runner.services)
	assert.NotNil(t, runner.notify)
}

func TestNewRunnerNotifyFlag(t *testing.T) {
	ctx := testutil.NewContext(testutil.WithNoNotify(false))
	runner := NewRunner(ctx, "main")
	notifyAdapter, ok := runner.notify.(*NotifyClientAdapter)
	require.True(t, ok)
	assert.True(t, notifyAdapter.shouldNotify)

	ctx2 := testutil.NewContext(testutil.WithNoNotify(true))
	runner2 := NewRunner(ctx2, "main")
	notifyAdapter2, ok := runner2.notify.(*NotifyClientAdapter)
	require.True(t, ok)
	assert.False(t, notifyAdapter2.shouldNotify)
}

func TestProjectClientAdapterAllRequirementsPassing(t *testing.T) {
	adapter := NewProjectClientAdapter()

	allPassing := &project.Project{
		Slug: "test",
		Requirements: []project.Requirement{
			{Slug: "a", Items: []string{"a"}, Passing: true},
			{Slug: "b", Items: []string{"b"}, Passing: true},
		},
	}
	assert.True(t, adapter.AllRequirementsPassing(allPassing))

	someFailing := &project.Project{
		Slug: "test",
		Requirements: []project.Requirement{
			{Slug: "a", Items: []string{"a"}, Passing: true},
			{Slug: "b", Items: []string{"b"}, Passing: false},
		},
	}
	assert.False(t, adapter.AllRequirementsPassing(someFailing))
}

func TestProjectClientAdapterMaxIterationsError(t *testing.T) {
	adapter := NewProjectClientAdapter()

	proj := &project.Project{
		Slug: "test",
		Requirements: []project.Requirement{
			{Slug: "a", Items: []string{"a"}, Passing: true},
			{Slug: "b", Items: []string{"b"}, Passing: false},
			{Slug: "c", Items: []string{"c"}, Passing: false},
		},
	}

	err := adapter.MaxIterationsError(proj)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMaxIterationsReached))
	assert.Contains(t, err.Error(), "2 requirements still failing")
}

func TestProjectClientAdapterMaxIterationsErrorAllPassing(t *testing.T) {
	adapter := NewProjectClientAdapter()

	proj := &project.Project{
		Slug: "test",
		Requirements: []project.Requirement{
			{Slug: "a", Items: []string{"a"}, Passing: true},
		},
	}

	err := adapter.MaxIterationsError(proj)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMaxIterationsReached))
	assert.Contains(t, err.Error(), "0 requirements still failing")
}

func TestAgentClientAdapterIsFatal(t *testing.T) {
	ctx := testutil.NewContext()
	adapter := NewAgentClientAdapter(ctx)

	assert.True(t, adapter.IsFatal(errors.New("Insufficient Balance")))
	assert.True(t, adapter.IsFatal(errors.New("billing error")))
	assert.True(t, adapter.IsFatal(errors.New("quota exceeded")))
	assert.True(t, adapter.IsFatal(errors.New("account issue")))
	assert.True(t, adapter.IsFatal(errors.New("payment required")))
	assert.False(t, adapter.IsFatal(errors.New("some other error")))
	assert.False(t, adapter.IsFatal(nil))
}

func TestGitClientAdapterBlockedFileInRepo(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	adapter := NewGitClientAdapter(testutil.NewContext())

	assert.False(t, adapter.BlockedFileExists())

	blockedPath := filepath.Join(workDir, "blocked.md")
	require.NoError(t, os.WriteFile(blockedPath, []byte("blocked"), 0644))
	t.Cleanup(func() { os.Remove(blockedPath) })

	assert.True(t, adapter.BlockedFileExists())
}

func TestGitClientAdapterWriteBlockedFileInRepo(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	adapter := NewGitClientAdapter(testutil.NewContext())
	adapter.WriteBlockedFile(errors.New("test error"))

	blockedPath := filepath.Join(workDir, "blocked.md")
	t.Cleanup(func() { os.Remove(blockedPath) })

	data, err := os.ReadFile(blockedPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test error")
}

func TestGitClientAdapterHasChanges(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	adapter := NewGitClientAdapter(testutil.NewContext())
	assert.False(t, adapter.HasChanges())

	require.NoError(t, os.WriteFile("new-test-file.txt", []byte("test"), 0644))
	t.Cleanup(func() { os.Remove("new-test-file.txt") })

	assert.True(t, adapter.HasChanges())
}

func TestGitClientAdapterReportExists(t *testing.T) {
	adapter := NewGitClientAdapter(testutil.NewContext())

	assert.False(t, adapter.ReportExists())

	require.NoError(t, os.WriteFile("report.md", []byte("test"), 0644))
	t.Cleanup(func() { os.Remove("report.md") })

	assert.True(t, adapter.ReportExists())
}

func TestNotifyClientAdapterError(t *testing.T) {
	enabled := NewNotifyClientAdapter(true)
	enabled.Error("test-slug")

	disabled := NewNotifyClientAdapter(false)
	disabled.Error("test-slug")
}

func TestNotifyClientAdapterSuccess(t *testing.T) {
	enabled := NewNotifyClientAdapter(true)
	enabled.Success("test-slug")

	disabled := NewNotifyClientAdapter(false)
	disabled.Success("test-slug")
}

func TestServicesClientAdapterRunBeforeCommandsEmpty(t *testing.T) {
	adapter := NewServicesClientAdapter()
	err := adapter.RunBeforeCommands(&config.RalphConfig{})
	assert.NoError(t, err)
}

func TestServicesClientAdapterRunBeforeCommandsNonEmpty(t *testing.T) {
	adapter := NewServicesClientAdapter()
	err := adapter.RunBeforeCommands(&config.RalphConfig{
		Before: []config.Before{
			{Name: "echo", Command: "echo", Args: []string{"hello"}},
		},
	})
	assert.NoError(t, err)
}
