package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
)

func TestCreateWorktreeDryRun(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	cmd, err := CreateWorktree("my-feature", true)
	require.NoError(t, err)

	expectedPath := filepath.Join(filepath.Dir(workDir), filepath.Base(workDir)+"-my-feature")
	assert.Equal(t, []string{"worktree", "add", "-b", "my-feature", expectedPath}, cmd.Args)
	assert.Equal(t, expectedPath, cmd.Path)
}

func TestCreateWorktreeDryRunDoesNotRunGit(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	cmd, err := CreateWorktree("my-feature", true)
	require.NoError(t, err, "dry-run succeeds without a git repository")
	assert.NotEmpty(t, cmd.Args)
	assert.NotEmpty(t, cmd.Path)
}

func TestBranchCheckedOutInWorktreeDryRun(t *testing.T) {
	t.Chdir(t.TempDir())

	cmd, checkedOut, err := BranchCheckedOutInWorktree("my-feature", true)
	require.NoError(t, err)
	assert.Equal(t, []string{"worktree", "list", "--porcelain"}, cmd.Args)
	assert.False(t, checkedOut)
}

func TestRemoveWorktreeDryRun(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	cmd, err := RemoveWorktree("my-feature", true)
	require.NoError(t, err)

	expectedPath := filepath.Join(filepath.Dir(workDir), filepath.Base(workDir)+"-my-feature")
	assert.Equal(t, []string{"worktree", "remove", "--force", expectedPath}, cmd.Args)
	assert.Equal(t, expectedPath, cmd.Path)
}

func TestGitClientWorktreeOperationsDryRun(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)

	client := NewClient(context.NewContext())

	t.Run("create", func(t *testing.T) {
		cmd, err := client.CreateWorktree("my-feature", true)
		require.NoError(t, err)
		expectedPath := filepath.Join(filepath.Dir(workDir), filepath.Base(workDir)+"-my-feature")
		assert.Equal(t, []string{"worktree", "add", "-b", "my-feature", expectedPath}, cmd.Args)
		assert.Equal(t, expectedPath, cmd.Path)
	})

	t.Run("detect", func(t *testing.T) {
		cmd, checkedOut, err := client.BranchCheckedOutInWorktree("my-feature", true)
		require.NoError(t, err)
		assert.Equal(t, []string{"worktree", "list", "--porcelain"}, cmd.Args)
		assert.False(t, checkedOut)
	})

	t.Run("remove", func(t *testing.T) {
		cmd, err := client.RemoveWorktree("my-feature", true)
		require.NoError(t, err)
		expectedPath := filepath.Join(filepath.Dir(workDir), filepath.Base(workDir)+"-my-feature")
		assert.Equal(t, []string{"worktree", "remove", "--force", expectedPath}, cmd.Args)
		assert.Equal(t, expectedPath, cmd.Path)
	})
}

func TestCreateWorktreeExistingBranch(t *testing.T) {
	workDir := setupTestRepo(t)
	t.Chdir(workDir)
	runGitForTest(t, workDir, "branch", "feature")

	cmd, err := CreateWorktree("feature", false)
	require.NoError(t, err)

	expectedPath := filepath.Join(filepath.Dir(workDir), filepath.Base(workDir)+"-feature")
	assert.Equal(t, expectedPath, cmd.Path)
	assert.Equal(t, []string{"worktree", "add", expectedPath, "feature"}, cmd.Args)
	assert.DirExists(t, expectedPath)

	worktreeList := runGitForTest(t, workDir, "worktree", "list", "--porcelain")
	assert.Contains(t, worktreeList, "branch refs/heads/feature")
	assert.Equal(t, "main", currentBranchForTest(t, workDir), "the current checkout stays on its branch")
}

func TestCreateWorktreeCreatesBranch(t *testing.T) {
	workDir := setupTestRepo(t)
	t.Chdir(workDir)

	cmd, err := CreateWorktree("brand-new", false)
	require.NoError(t, err)

	expectedPath := filepath.Join(filepath.Dir(workDir), filepath.Base(workDir)+"-brand-new")
	assert.Equal(t, []string{"worktree", "add", "-b", "brand-new", expectedPath}, cmd.Args)
	assert.DirExists(t, expectedPath)

	worktreeList := runGitForTest(t, workDir, "worktree", "list", "--porcelain")
	assert.Contains(t, worktreeList, "branch refs/heads/brand-new")
	assert.Equal(t, "main", currentBranchForTest(t, workDir), "the current checkout stays on its branch")
}

func TestBranchCheckedOutInWorktree(t *testing.T) {
	workDir := setupTestRepo(t)
	t.Chdir(workDir)
	runGitForTest(t, workDir, "branch", "feature")
	_, err := CreateWorktree("feature", false)
	require.NoError(t, err)

	_, checkedOut, err := BranchCheckedOutInWorktree("feature", false)
	require.NoError(t, err)
	assert.True(t, checkedOut, "a branch checked out in another worktree is detected")

	_, checkedOut, err = BranchCheckedOutInWorktree("main", false)
	require.NoError(t, err)
	assert.True(t, checkedOut, "the current branch counts as checked out")

	_, checkedOut, err = BranchCheckedOutInWorktree("missing", false)
	require.NoError(t, err)
	assert.False(t, checkedOut, "a branch checked out nowhere is not detected")
}

func TestRemoveWorktree(t *testing.T) {
	workDir := setupTestRepo(t)
	t.Chdir(workDir)
	runGitForTest(t, workDir, "branch", "feature")
	worktreePath, err := CreateWorktree("feature", false)
	require.NoError(t, err)

	_, err = RemoveWorktree("feature", false)
	require.NoError(t, err)

	_, err = os.Stat(worktreePath.Path)
	assert.True(t, os.IsNotExist(err), "the worktree directory is removed")
	worktreeList := runGitForTest(t, workDir, "worktree", "list", "--porcelain")
	assert.NotContains(t, worktreeList, "branch refs/heads/feature")
}

func runGitForTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

func currentBranchForTest(t *testing.T, dir string) string {
	t.Helper()
	return runGitForTest(t, dir, "rev-parse", "--abbrev-ref", "HEAD")
}
