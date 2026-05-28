package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/testutil"
)

func TestGitRunAdapterNewAdapter(t *testing.T) {
	ctx := context.NewContext()
	adapter := git.NewRunAdapter(ctx)
	require.NotNil(t, adapter)
	var _ orchestrationRun.GitClient = adapter
}

func TestGitRunAdapterBlockedFileExists(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	adapter := git.NewRunAdapter(context.NewContext())

	t.Run("returns false when no blocked.md exists", func(t *testing.T) {
		assert.False(t, adapter.BlockedFileExists())
	})

	t.Run("returns true when blocked.md exists in repo root", func(t *testing.T) {
		blockedPath := filepath.Join(workDir, "blocked.md")
		require.NoError(t, os.WriteFile(blockedPath, []byte("blocked"), 0644))
		assert.True(t, adapter.BlockedFileExists())
	})
}

func TestGitRunAdapterWriteBlockedFile(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	adapter := git.NewRunAdapter(context.NewContext())
	err := &testBlockedError{"connection refused"}

	adapter.WriteBlockedFile(err)

	blockedPath := filepath.Join(workDir, "blocked.md")
	data, readErr := os.ReadFile(blockedPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "connection refused")
	assert.Contains(t, string(data), "# Blocked")
}

func TestGitRunAdapterHasChanges(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	adapter := git.NewRunAdapter(context.NewContext())

	t.Run("returns false with clean working tree", func(t *testing.T) {
		assert.False(t, adapter.HasChanges())
	})

	t.Run("returns true after modifying a file", func(t *testing.T) {
		require.NoError(t, os.WriteFile("new.txt", []byte("content"), 0644))
		assert.True(t, adapter.HasChanges())
	})
}

func TestGitRunAdapterReportExists(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	adapter := git.NewRunAdapter(context.NewContext())

	t.Run("returns false when no report.md exists", func(t *testing.T) {
		assert.False(t, adapter.ReportExists())
	})

	t.Run("returns true when report.md exists", func(t *testing.T) {
		require.NoError(t, os.WriteFile("report.md", []byte("report content"), 0644))
		assert.True(t, adapter.ReportExists())
	})
}

func TestGitRunAdapterCommitFromReport(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	ctx := context.NewContext()
	adapter := git.NewRunAdapter(ctx)

	reportContent := "Implement requirement: adapter-git"
	require.NoError(t, os.WriteFile("report.md", []byte(reportContent), 0644))
	require.NoError(t, os.WriteFile("newfile.txt", []byte("change"), 0644))

	err := adapter.CommitFromReport("test-slug")
	require.NoError(t, err)

	_, err = os.Stat("report.md")
	assert.True(t, os.IsNotExist(err), "report.md should be deleted after commit")
}

func TestGitRunAdapterCommitFromReportFailsWhenNoReport(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	adapter := git.NewRunAdapter(context.NewContext())

	err := adapter.CommitFromReport("test-slug")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report.md")
}

func setupLocalRemote(t *testing.T, dir string) {
	t.Helper()
	bareDir := t.TempDir()

	c := exec.Command("git", "init", "--bare", bareDir)
	c.Dir = dir
	require.NoError(t, c.Run())

	c = exec.Command("git", "remote", "add", "origin", bareDir)
	c.Dir = dir
	require.NoError(t, c.Run())

	c = exec.Command("git", "push", "--set-upstream", "origin", "main")
	c.Dir = dir
	require.NoError(t, c.Run())
}

type testBlockedError struct {
	msg string
}

func (e *testBlockedError) Error() string {
	return e.msg
}
