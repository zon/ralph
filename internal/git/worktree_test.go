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

func TestCreateWorktreeStartsFromRemoteTip(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)
	base := currentBranchForTest(t, workDir)

	// Build the project branch on the remote, then drop the local branch, as a
	// run in another checkout leaves it.
	require.NoError(t, CheckoutOrCreateBranch("project"))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "remote-work.txt"), []byte("remote\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("add remote work"))
	_, err := Push(nil, "project")
	require.NoError(t, err)
	remoteTip := runGitForTest(t, workDir, "rev-parse", "project")
	require.NoError(t, CheckoutBranch(base))
	runGitForTest(t, workDir, "branch", "-D", "project")

	cmd, err := CreateWorktree("project", false)
	require.NoError(t, err)

	assert.Equal(t, []string{"worktree", "add", "-b", "project", cmd.Path, "origin/project"}, cmd.Args)
	assert.Equal(t, remoteTip, runGitForTest(t, cmd.Path, "rev-parse", "HEAD"), "the worktree starts from the remote tip")
	assert.FileExists(t, filepath.Join(cmd.Path, "remote-work.txt"), "the worktree holds the remote's work")

	_, err = RemoveWorktree("project", false)
	require.NoError(t, err)
}

func TestCreateWorktreeFastForwardsLocalBranch(t *testing.T) {
	workDir, remoteDir := setupBareRemoteRepo(t)
	t.Chdir(workDir)
	base := currentBranchForTest(t, workDir)

	require.NoError(t, CheckoutOrCreateBranch("project"))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "first.txt"), []byte("first\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("add first work"))
	_, err := Push(nil, "project")
	require.NoError(t, err)
	require.NoError(t, CheckoutBranch(base))

	remoteTip := advanceRemoteBranch(t, remoteDir, "project", "advance.txt")

	cmd, err := CreateWorktree("project", false)
	require.NoError(t, err)

	assert.Equal(t, remoteTip, runGitForTest(t, workDir, "rev-parse", "project"), "the local branch moves to the remote tip")
	assert.Equal(t, remoteTip, runGitForTest(t, cmd.Path, "rev-parse", "HEAD"))
	assert.FileExists(t, filepath.Join(cmd.Path, "advance.txt"))

	_, err = RemoveWorktree("project", false)
	require.NoError(t, err)
}

func TestCreateWorktreeKeepsLocalCommitsAheadOfRemote(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)
	base := currentBranchForTest(t, workDir)

	require.NoError(t, CheckoutOrCreateBranch("project"))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "pushed.txt"), []byte("pushed\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("add pushed work"))
	_, err := Push(nil, "project")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "unpushed.txt"), []byte("unpushed\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("add unpushed work"))
	localTip := runGitForTest(t, workDir, "rev-parse", "HEAD")
	require.NoError(t, CheckoutBranch(base))

	cmd, err := CreateWorktree("project", false)
	require.NoError(t, err)

	assert.Equal(t, localTip, runGitForTest(t, cmd.Path, "rev-parse", "HEAD"), "local commits the remote lacks are kept")

	_, err = RemoveWorktree("project", false)
	require.NoError(t, err)
}

func TestCreateWorktreeRefusesDivergedBranch(t *testing.T) {
	workDir, remoteDir := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	require.NoError(t, CheckoutOrCreateBranch("project"))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "shared.txt"), []byte("shared\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("add shared work"))
	_, err := Push(nil, "project")
	require.NoError(t, err)

	advanceRemoteBranch(t, remoteDir, "project", "advance.txt")
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "local.txt"), []byte("local\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("add local work"))

	_, err = CreateWorktree("project", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "diverged")
	assert.NoDirExists(t, filepath.Join(filepath.Dir(workDir), filepath.Base(workDir)+"-project"), "no worktree is created for a diverged branch")
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
