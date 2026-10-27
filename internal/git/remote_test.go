package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPush_HappyPath verifies that Push succeeds against a local
// bare remote. This is the happy-path integration test: no network access or
// GitHub credentials are needed because the remote is a local file-system path.
func TestPush_HappyPath(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	// Create a new feature branch with a commit to push.
	branchName := "feature/push-test"
	require.NoError(t, CheckoutOrCreateBranch(branchName))

	if err := os.WriteFile(filepath.Join(workDir, "feature.txt"), []byte("feature\n"), 0644); err != nil {
		t.Fatalf("failed to create feature file: %v", err)
	}

	require.NoError(t, StageAll())
	require.NoError(t, Commit("add feature"))

	remoteURL, err := Push(nil, branchName)
	require.NoError(t, err, "Push failed")
	assert.NotEmpty(t, remoteURL, "Push returned an empty remote URL")
}

func TestIsWorkflowPermissionError(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "GitHub App workflow rejection message",
			output:   "! [remote rejected] ci-container-build -> ci-container-build (refusing to allow a GitHub App to create or update workflow `.github/workflows/container-build.yaml` without `workflows` permission)\nerror: failed to push some refs",
			expected: true,
		},
		{
			name:     "permission fragment",
			output:   "without `workflows` permission",
			expected: true,
		},
		{
			name:     "OAuth App workflow rejection message",
			output:   "! [remote rejected] ci-container-build -> ci-container-build (refusing to allow an OAuth App to create or update workflow `.github/workflows/container-build.yaml` without `workflow` scope)\nerror: failed to push some refs",
			expected: true,
		},
		{
			name:     "scope fragment",
			output:   "without `workflow` scope",
			expected: true,
		},
		{
			name:     "regular push failure",
			output:   "error: failed to push some refs to 'https://github.com/foo/bar.git'",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWorkflowPermissionError(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPush_WorkflowPermissionError(t *testing.T) {
	// Set up a local bare remote that rejects pushes with the GitHub workflow
	// permission message.
	remoteDir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

	// Write a pre-receive hook that mimics GitHub's workflow-permission rejection.
	hookContent := "#!/bin/sh\necho 'refusing to allow a GitHub App to create or update workflow `.github/workflows/test.yaml` without `workflows` permission' >&2\nexit 1\n"
	hookPath := filepath.Join(remoteDir, "hooks", "pre-receive")
	require.NoError(t, os.WriteFile(hookPath, []byte(hookContent), 0755))

	// Clone the bare remote into a working copy.
	workDir := t.TempDir()
	require.NoError(t, exec.Command("git", "clone", remoteDir, workDir).Run())

	t.Chdir(workDir)

	// Configure identity
	_ = exec.Command("git", "config", "--local", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "config", "--local", "user.name", "Test User").Run()

	// Create a workflow file commit
	wfDir := filepath.Join(workDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(wfDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "test.yaml"), []byte("name: test\n"), 0644))

	require.NoError(t, StageAll())
	require.NoError(t, Commit("Add workflow file"))

	branch, _ := GetCurrentBranch()
	_, pushErr := Push(nil, branch)
	require.Error(t, pushErr)
	assert.True(t, errors.Is(pushErr, ErrWorkflowPermission))
}

func TestPullRebase_WithNewCommits(t *testing.T) {
	remoteDir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

	workDir1 := t.TempDir()
	require.NoError(t, exec.Command("git", "clone", remoteDir, workDir1).Run())

	// Setup workDir1
	_ = exec.Command("git", "-C", workDir1, "config", "--local", "user.email", "test1@example.com").Run()
	_ = exec.Command("git", "-C", workDir1, "config", "--local", "user.name", "Test 1").Run()

	branchName := "pull-test-branch"
	_ = exec.Command("git", "-C", workDir1, "checkout", "-b", branchName).Run()
	os.WriteFile(workDir1+"/test.txt", []byte("test1\n"), 0644)
	_ = exec.Command("git", "-C", workDir1, "add", ".").Run()
	_ = exec.Command("git", "-C", workDir1, "commit", "-m", "commit 1").Run()
	_ = exec.Command("git", "-C", workDir1, "push", "origin", branchName).Run()

	// Setup workDir2 and push another commit
	workDir2 := t.TempDir()
	require.NoError(t, exec.Command("git", "clone", remoteDir, workDir2).Run())
	_ = exec.Command("git", "-C", workDir2, "checkout", branchName).Run()
	_ = exec.Command("git", "-C", workDir2, "config", "--local", "user.email", "test2@example.com").Run()
	_ = exec.Command("git", "-C", workDir2, "config", "--local", "user.name", "Test 2").Run()

	os.WriteFile(workDir2+"/test2.txt", []byte("test2\n"), 0644)
	_ = exec.Command("git", "-C", workDir2, "add", ".").Run()
	_ = exec.Command("git", "-C", workDir2, "commit", "-m", "commit 2").Run()
	_ = exec.Command("git", "-C", workDir2, "push", "origin", branchName).Run()

	// Back to workDir1 and pull rebase
	t.Chdir(workDir1)
	require.NoError(t, PullRebase(nil))
}

func TestPullRebase_CommitsDirtyTreeBeforePull(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	branchName := "feature/dirty-pull"
	require.NoError(t, CheckoutOrCreateBranch(branchName))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "feature.txt"), []byte("feature\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("add feature"))
	_, err := Push(nil, branchName)
	require.NoError(t, err)

	// Leave the worktree dirty after the last commit, as a straggling agent
	// write would. The pull before push must sweep it into a commit, not abort.
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "feature.txt"), []byte("feature\nmore\n"), 0644))
	require.True(t, HasUncommittedChanges())

	require.NoError(t, PullAndPush(false, "", ""))
	require.False(t, HasUncommittedChanges())

	messages, err := CommitMessages("HEAD~1")
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, workingTreeCommitMessage+"\n", messages[0])
}

func TestClone(t *testing.T) {
	remoteDir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

	// Need at least one commit for HEAD to exist
	workDir := t.TempDir()
	_ = exec.Command("git", "clone", remoteDir, workDir).Run()
	_ = exec.Command("git", "-C", workDir, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", workDir, "config", "user.name", "test").Run()
	os.WriteFile(workDir+"/README.md", []byte("# test"), 0644)
	_ = exec.Command("git", "-C", workDir, "add", ".").Run()
	_ = exec.Command("git", "-C", workDir, "commit", "-m", "init").Run()
	_ = exec.Command("git", "-C", workDir, "push", "origin", "HEAD").Run()

	cloneDir := t.TempDir()
	err := Clone(remoteDir, "", cloneDir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(cloneDir, ".git"))
	assert.NoError(t, err)
}

func TestRemoteURL(t *testing.T) {
	t.Run("returns remote URL from config", func(t *testing.T) {
		workDir, remoteDir := setupBareRemoteRepo(t)
		t.Chdir(workDir)

		remoteURL, err := RemoteURL()
		require.NoError(t, err)
		assert.Equal(t, remoteDir, remoteURL)
	})

	t.Run("returns error when no remote origin", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Chdir(tempDir)

		cmd := exec.Command("git", "init")
		require.NoError(t, cmd.Run())

		_, err := RemoteURL()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get remote URL")
	})
}

func TestRebaseInProgress(t *testing.T) {
	tests := []struct {
		name string
		dir  string
	}{
		{name: "no rebase directory"},
		{name: "merge backend", dir: "rebase-merge"},
		{name: "apply backend", dir: "rebase-apply"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := setupTestRepo(t)
			t.Chdir(workDir)

			if tt.dir == "" {
				assert.False(t, rebaseInProgress())
				return
			}
			require.NoError(t, os.MkdirAll(filepath.Join(workDir, ".git", tt.dir), 0755))
			assert.True(t, rebaseInProgress())
		})
	}
}

// TestPullAndPush_AbortsConflictingRebase verifies that a pull which stops on a
// conflict is aborted, so the run fails with the local commit and a clean
// checkout instead of conflict markers and a half-applied rebase.
func TestPullAndPush_AbortsConflictingRebase(t *testing.T) {
	workDir, remoteDir := setupBareRemoteRepo(t)
	t.Chdir(workDir)
	branch := currentBranchForTest(t, workDir)

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "shared.txt"), []byte("one\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("add shared file"))
	_, err := Push(nil, branch)
	require.NoError(t, err)

	// Advance the remote from another checkout so both sides edit the same line.
	other := t.TempDir()
	out, err := exec.Command("git", "clone", remoteDir, other).CombinedOutput()
	require.NoError(t, err, "git clone: %s", out)
	runGitForTest(t, other, "config", "user.email", "test@example.com")
	runGitForTest(t, other, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(other, "shared.txt"), []byte("two\n"), 0644))
	runGitForTest(t, other, "add", ".")
	runGitForTest(t, other, "commit", "-m", "remote edit")
	runGitForTest(t, other, "push", "origin", branch)

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "shared.txt"), []byte("three\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("local edit"))
	localTip := runGitForTest(t, workDir, "rev-parse", "HEAD")

	err = PullAndPush(false, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to pull before push")
	assert.False(t, rebaseInProgress(), "the conflicting rebase is aborted")

	assert.Equal(t, localTip, runGitForTest(t, workDir, "rev-parse", "HEAD"), "the local commit survives the abort")
	content, err := os.ReadFile(filepath.Join(workDir, "shared.txt"))
	require.NoError(t, err)
	assert.Equal(t, "three\n", string(content), "the working tree holds the local edit, not conflict markers")
	assert.Empty(t, runGitForTest(t, workDir, "status", "--porcelain"), "the checkout is clean")
}

// TestPullRebase_LeavesRunningRebaseAlone verifies that a rebase which was
// already running before the pull is not aborted by the pull's cleanup.
func TestPullRebase_LeavesRunningRebaseAlone(t *testing.T) {
	workDir, remoteDir := setupBareRemoteRepo(t)
	t.Chdir(workDir)
	branch := currentBranchForTest(t, workDir)

	// Make the pull rebase local work onto a moved remote, then leave a rebase
	// that started before the pull in place.
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "local.txt"), []byte("local\n"), 0644))
	require.NoError(t, StageAll())
	require.NoError(t, Commit("add local work"))
	advanceRemoteBranch(t, remoteDir, branch, "advance.txt")
	rebaseDir := filepath.Join(workDir, ".git", "rebase-merge")
	require.NoError(t, os.MkdirAll(rebaseDir, 0755))

	require.Error(t, PullRebase(nil))
	assert.DirExists(t, rebaseDir, "a rebase that was already running is left alone")
}

func TestFetchBranch(t *testing.T) {
	t.Run("fetches branch with refspec", func(t *testing.T) {
		workDir, _ := setupBareRemoteRepo(t)
		t.Chdir(workDir)

		branchName := "fetch-test-branch"
		require.NoError(t, CheckoutOrCreateBranch(branchName))

		require.NoError(t, os.WriteFile(filepath.Join(workDir, "test.txt"), []byte("test\n"), 0644))
		require.NoError(t, StageAll())
		require.NoError(t, Commit("add test file"))
		_, err := Push(nil, branchName)
		require.NoError(t, err)

		require.NoError(t, FetchBranch(branchName))

		refs, err := runGit("rev-parse", "--verify", branchName)
		require.NoError(t, err)
		assert.NotEmpty(t, refs)
	})

	t.Run("falls back to plain fetch when refspec fails", func(t *testing.T) {
		workDir, _ := setupBareRemoteRepo(t)
		t.Chdir(workDir)

		branchName := "fetch-fallback-branch"
		require.NoError(t, CheckoutOrCreateBranch(branchName))

		require.NoError(t, os.WriteFile(filepath.Join(workDir, "test.txt"), []byte("test\n"), 0644))
		require.NoError(t, StageAll())
		require.NoError(t, Commit("add test file"))
		_, err := Push(nil, branchName)
		require.NoError(t, err)

		require.NoError(t, FetchBranch(branchName))

		refs, err := runGit("rev-parse", "--verify", branchName)
		require.NoError(t, err)
		assert.NotEmpty(t, refs)
	})

	t.Run("returns error when fallback fetch also fails", func(t *testing.T) {
		workDir, _ := setupBareRemoteRepo(t)
		t.Chdir(workDir)

		err := FetchBranch("nonexistent-branch")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch branch")
	})
}
