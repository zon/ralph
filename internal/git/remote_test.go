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
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(tempDir)

		cmd := exec.Command("git", "init")
		require.NoError(t, cmd.Run())

		_, err := RemoteURL()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get remote URL")
	})
}
