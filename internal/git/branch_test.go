package git

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	branch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")

	// Default branch should be 'master' or 'main'
	assert.True(t, branch == "master" || branch == "main", "Expected branch to be 'master' or 'main', got '%s'", branch)
}

func TestGetCurrentBranch_DetachedHead(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Get the commit hash to checkout
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tempDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	commitHash := string(output[:7]) // Use first 7 chars

	// Checkout the commit directly (detached HEAD)
	cmd = exec.Command("git", "checkout", commitHash)
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to checkout commit: %v", err)
	}

	// GetCurrentBranch should return error for detached HEAD
	_, err = GetCurrentBranch()
	require.Error(t, err, "Expected GetCurrentBranch to return error in detached HEAD state")

	expectedMsg := "detached HEAD state"
	assert.Contains(t, err.Error(), expectedMsg, "Expected error containing '%s', got: %v", expectedMsg, err)
}

func TestCheckoutOrCreateBranch_CreateNew(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	branchName := "brand-new-branch"

	require.NoError(t, CheckoutOrCreateBranch(branchName), "CheckoutOrCreateBranch failed")

	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")

	assert.Equal(t, branchName, currentBranch)
}

func TestHasCommits(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Should have commits (setupTestRepo creates an initial commit)
	assert.True(t, hasCommits(), "Expected hasCommits to return true for repo with commits")
}

func TestCheckoutOrCreateBranch_ExistingRemoteBranch(t *testing.T) {
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	workDir := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v\n%s", err, out)
	}

	// Configure identity
	_ = exec.Command("git", "-C", workDir, "config", "--local", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", workDir, "config", "--local", "user.name", "Test User").Run()

	// Initial commit
	os.WriteFile(workDir+"/README.md", []byte("# test\n"), 0644)
	_ = exec.Command("git", "-C", workDir, "add", ".").Run()
	_ = exec.Command("git", "-C", workDir, "commit", "-m", "initial commit").Run()
	_ = exec.Command("git", "-C", workDir, "push", "origin", "HEAD").Run()

	branchName := "existing-remote-branch"
	_ = exec.Command("git", "-C", workDir, "checkout", "-b", branchName).Run()
	os.WriteFile(workDir+"/test.txt", []byte("test\n"), 0644)
	_ = exec.Command("git", "-C", workDir, "add", ".").Run()
	_ = exec.Command("git", "-C", workDir, "commit", "-m", "add test file").Run()
	_ = exec.Command("git", "-C", workDir, "push", "origin", branchName).Run()

	// Second clone
	workDir2 := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir2)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to clone: %v\n%s", err, out)
	}

	t.Chdir(workDir2)
	require.NoError(t, CheckoutOrCreateBranch(branchName), "CheckoutOrCreateBranch failed")

	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")
	assert.Equal(t, branchName, currentBranch)
}

func TestIsBranchSyncedWithRemote_Synced(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	branchName := "synced-branch"
	_ = exec.Command("git", "checkout", "-b", branchName).Run()
	os.WriteFile(workDir+"/test.txt", []byte("test\n"), 0644)
	_ = exec.Command("git", "add", ".").Run()
	_ = exec.Command("git", "commit", "-m", "add test file").Run()
	_ = exec.Command("git", "push", "origin", branchName).Run()

	require.NoError(t, IsBranchSyncedWithRemote(branchName), "IsBranchSyncedWithRemote failed for synced branch")
}
