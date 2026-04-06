package git

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name         string
		projectName  string
		expectedName string
	}{
		{
			name:         "simple name",
			projectName:  "fix-pagination",
			expectedName: "fix-pagination",
		},
		{
			name:         "spaces in name",
			projectName:  "my cool feature",
			expectedName: "my-cool-feature",
		},
		{
			name:         "uppercase letters",
			projectName:  "MyFeature",
			expectedName: "myfeature",
		},
		{
			name:         "underscores",
			projectName:  "my_feature_branch",
			expectedName: "my-feature-branch",
		},
		{
			name:         "special characters",
			projectName:  "my@feature!",
			expectedName: "myfeature",
		},
		{
			name:         "multiple dots",
			projectName:  "my.feature.name",
			expectedName: "my-feature-name",
		},
		{
			name:         "leading/trailing hyphens",
			projectName:  "-my-feature-",
			expectedName: "my-feature",
		},
		{
			name:         "consecutive hyphens",
			projectName:  "my--feature",
			expectedName: "my-feature",
		},
		{
			name:         "empty after sanitization",
			projectName:  "---",
			expectedName: "unnamed-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeBranchName(tt.projectName)
			assert.Equal(t, tt.expectedName, got, "SanitizeBranchName should return expected value")
		})
	}
}

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

func TestValidateGitStateAndSwitchBranch_AlreadyOnBranch(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	ctx := context.NewContext()
	ctx.SetLocal(true)

	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")

	err = ValidateGitStateAndSwitchBranch(ctx, currentBranch)
	require.NoError(t, err, "ValidateGitStateAndSwitchBranch failed")

	newBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")
	assert.Equal(t, currentBranch, newBranch, "Should stay on the same branch")
}

func TestValidateGitStateAndSwitchBranch_SwitchToNewBranch(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	ctx := context.NewContext()
	ctx.SetLocal(true)

	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")

	newBranchName := "my-feature-branch"
	require.NotEqual(t, currentBranch, newBranchName, "Test setup error: branches should be different")

	err = ValidateGitStateAndSwitchBranch(ctx, newBranchName)
	require.NoError(t, err, "ValidateGitStateAndSwitchBranch failed")

	newBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")
	assert.Equal(t, newBranchName, newBranch, "Should have switched to new branch")
}

func TestValidateGitStateAndSwitchBranch_WorkflowContext(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	ctx := context.NewContext()
	ctx.SetLocal(true)
	ctx.SetWorkflowExecution(true)
	ctx.SetRepoOwner("testowner")
	ctx.SetRepoName("testrepo")

	newBranchName := "workflow-test-branch"
	err := ValidateGitStateAndSwitchBranch(ctx, newBranchName)
	require.NoError(t, err, "ValidateGitStateAndSwitchBranch failed in workflow context")

	newBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")
	assert.Equal(t, newBranchName, newBranch, "Should have switched to new branch in workflow context")
}

func TestSwitchToProjectBranch_CreatesNewBranch(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	ctx := context.NewContext()
	ctx.SetLocal(true)

	branchName := "brand-new-branch"
	err := SwitchToProjectBranch(ctx, branchName)
	require.NoError(t, err, "SwitchToProjectBranch failed")

	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")
	assert.Equal(t, branchName, currentBranch, "Should be on the new branch")
}

func TestSwitchToProjectBranch_ExitingRemoteBranch(t *testing.T) {
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

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test@example.com"},
		{"config", "--local", "user.name", "Test User"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	os.WriteFile(workDir+"/README.md", []byte("# test\n"), 0644)
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "initial commit"},
		{"push", "origin", "HEAD"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	branchName := "remote-branch"
	_ = exec.Command("git", "-C", workDir, "checkout", "-b", branchName).Run()
	os.WriteFile(workDir+"/test.txt", []byte("test\n"), 0644)
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "add test file"},
		{"push", "origin", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	t.Chdir(workDir)

	ctx := context.NewContext()
	ctx.SetLocal(true)

	err := SwitchToProjectBranch(ctx, branchName)
	require.NoError(t, err, "SwitchToProjectBranch failed for existing remote branch")

	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")
	assert.Equal(t, branchName, currentBranch, "Should be on the remote branch")
}
