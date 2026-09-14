package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorktreeBaseBranchSyncLeavesCheckoutUntouched verifies that fetching and
// merging the base branch from inside a worktree brings the worktree branch up
// to date with the remote base tip while leaving the base branch checked out in
// the main checkout, and its working tree, untouched.
func TestWorktreeBaseBranchSyncLeavesCheckoutUntouched(t *testing.T) {
	workDir, remoteDir := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)
	baseBefore := runGitForTest(t, workDir, "rev-parse", base)

	_, err = CreateWorktree("project-branch", false)
	require.NoError(t, err)
	worktreePath := worktreePathFor(workDir, "project-branch")

	advanceRemoteBranch(t, remoteDir, base, "base-advance.txt")

	t.Chdir(worktreePath)
	require.NoError(t, FetchBranch(base))

	localNeedsMerge, err := NeedsMerge(base)
	require.NoError(t, err)
	require.False(t, localNeedsMerge, "the base branch checked out in the main checkout cannot be advanced, so the stale local ref sees no merge")

	needsMerge, err := NeedsMerge("origin/" + base)
	require.NoError(t, err)
	require.True(t, needsMerge, "the worktree branch is behind the advanced remote base branch")

	require.NoError(t, Merge("origin/"+base))
	assert.FileExists(t, filepath.Join(worktreePath, "base-advance.txt"), "the worktree branch contains the merged base change")

	t.Chdir(workDir)
	assert.Equal(t, baseBefore, runGitForTest(t, workDir, "rev-parse", base), "the base branch checked out in the main checkout is untouched")
	assert.NoFileExists(t, filepath.Join(workDir, "base-advance.txt"), "the main checkout working tree is untouched")

	_, err = RemoveWorktree("project-branch", false)
	require.NoError(t, err)
}

// advanceRemoteBranch clones the bare remote into a temporary directory, adds a
// commit on branch, and pushes it back, so the remote base tip advances without
// touching the working checkout.
func advanceRemoteBranch(t *testing.T, remoteDir, branch, filename string) {
	t.Helper()
	dir := t.TempDir()
	c := exec.Command("git", "clone", remoteDir, dir)
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git clone: %s", out)

	runGitForTest(t, dir, "config", "user.email", "test@example.com")
	runGitForTest(t, dir, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte("advance\n"), 0644))
	runGitForTest(t, dir, "add", ".")
	runGitForTest(t, dir, "commit", "-m", "advance base")
	runGitForTest(t, dir, "push", "origin", branch)
}
