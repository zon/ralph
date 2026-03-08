package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/testutil"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user (required for commits) - using --local to ensure isolation
	cmd = exec.Command("git", "config", "--local", "user.email", "test@example.com")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "config", "--local", "user.name", "Test User")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tempDir
}

func TestIsGitRepository(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Should be true inside a git repository
	if !IsGitRepository(ctx) {
		t.Error("Expected IsGitRepository to return true inside a git repo")
	}
}

func TestIsGitRepository_NotRepo(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Should be false outside a git repository
	if IsGitRepository(ctx) {
		t.Error("Expected IsGitRepository to return false outside a git repo")
	}
}

func TestIsGitRepository_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	// Should always return true in dry-run mode
	if !IsGitRepository(ctx) {
		t.Error("Expected IsGitRepository to return true in dry-run mode")
	}
}

func TestIsDetachedHead(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Should not be detached on a normal branch
	isDetached, err := IsDetachedHead(ctx)
	if err != nil {
		t.Fatalf("IsDetachedHead failed: %v", err)
	}

	if isDetached {
		t.Error("Expected IsDetachedHead to return false on a branch")
	}
}

func TestIsDetachedHead_Detached(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

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

	// Should be detached now
	isDetached, err := IsDetachedHead(ctx)
	if err != nil {
		t.Fatalf("IsDetachedHead failed: %v", err)
	}

	if !isDetached {
		t.Error("Expected IsDetachedHead to return true in detached HEAD state")
	}
}

func TestIsDetachedHead_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	// Should always return false in dry-run mode
	isDetached, err := IsDetachedHead(ctx)
	if err != nil {
		t.Fatalf("IsDetachedHead in dry-run failed: %v", err)
	}

	if isDetached {
		t.Error("Expected IsDetachedHead to return false in dry-run mode")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	branch, err := GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	// Default branch should be 'master' or 'main'
	if branch != "master" && branch != "main" {
		t.Errorf("Expected branch to be 'master' or 'main', got '%s'", branch)
	}
}

func TestGetCurrentBranch_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	branch, err := GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("GetCurrentBranch in dry-run failed: %v", err)
	}

	if branch != "dry-run-branch" {
		t.Errorf("Expected dry-run branch to be 'dry-run-branch', got '%s'", branch)
	}
}

func TestGetCurrentBranch_DetachedHead(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

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
	_, err = GetCurrentBranch(ctx)
	if err == nil {
		t.Error("Expected GetCurrentBranch to return error in detached HEAD state")
	}

	expectedMsg := "detached HEAD state"
	if err != nil && !testutil.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedMsg, err)
	}
}

func TestFetch_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	if err := Fetch(ctx); err != nil {
		t.Fatalf("Fetch in dry-run failed: %v", err)
	}
}

func TestRemoteBranchExists_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	exists := RemoteBranchExists(ctx, "any-branch")
	if exists {
		t.Error("Expected dry-run RemoteBranchExists to return false")
	}
}

func TestCheckoutOrCreateBranch_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	if err := CheckoutOrCreateBranch(ctx, "new-branch"); err != nil {
		t.Fatalf("CheckoutOrCreateBranch in dry-run failed: %v", err)
	}
}

func TestCheckoutOrCreateBranch_CreateNew(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	branchName := "brand-new-branch"

	if err := CheckoutOrCreateBranch(ctx, branchName); err != nil {
		t.Fatalf("CheckoutOrCreateBranch failed: %v", err)
	}

	currentBranch, err := GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != branchName {
		t.Errorf("Expected current branch to be '%s', got '%s'", branchName, currentBranch)
	}
}

func TestCheckoutBranch_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	if err := CheckoutBranch(ctx, "any-branch"); err != nil {
		t.Fatalf("CheckoutBranch in dry-run failed: %v", err)
	}
}

func TestHasCommits(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Should have commits (setupTestRepo creates an initial commit)
	if !HasCommits(ctx) {
		t.Error("Expected HasCommits to return true for repo with commits")
	}
}

func TestHasCommits_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	// In dry-run mode, should always return true
	if !HasCommits(ctx) {
		t.Error("Expected HasCommits to return true in dry-run mode")
	}
}

func TestPushBranch_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	// Should not return an error in dry-run mode
	url, err := PushBranch(ctx, "test-branch")
	if err != nil {
		t.Fatalf("PushBranch in dry-run failed: %v", err)
	}

	if url == "" {
		t.Error("Expected PushBranch to return a URL in dry-run mode")
	}
}

// setupBareRemoteRepo creates a temporary bare remote and a clone of it,
// configures git identity, makes an initial commit, pushes it, and returns
// (workDir, remoteDir). The caller must chdir into workDir before calling any
// git functions that rely on the working directory.
func setupBareRemoteRepo(t *testing.T) (workDir, remoteDir string) {
	t.Helper()

	remoteDir = t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	workDir = t.TempDir()
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

	// Create and push an initial commit so HEAD exists on the remote.
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}
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

	return workDir, remoteDir
}

// TestPushBranch_HappyPath verifies that PushBranch succeeds against a local
// bare remote. This is the happy-path integration test: no network access or
// GitHub credentials are needed because the remote is a local file-system path.
func TestPushBranch_HappyPath(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)

	// Create a new feature branch with a commit to push.
	branchName := "feature/push-test"
	for _, args := range [][]string{
		{"checkout", "-b", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(workDir, "feature.txt"), []byte("feature\n"), 0644); err != nil {
		t.Fatalf("failed to create feature file: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "add feature"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	t.Chdir(workDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	remoteURL, err := PushBranch(ctx, branchName)
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}
	if remoteURL == "" {
		t.Error("PushBranch returned an empty remote URL")
	}
}

// TestPushCurrentBranch_HappyPath verifies that PushCurrentBranch succeeds
// against a local bare remote.
func TestPushCurrentBranch_HappyPath(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)

	branchName := "feature/push-current-test"
	for _, args := range [][]string{
		{"checkout", "-b", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(workDir, "current.txt"), []byte("current\n"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "add current"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	t.Chdir(workDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	if err := PushCurrentBranch(ctx); err != nil {
		t.Fatalf("PushCurrentBranch failed: %v", err)
	}
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
			name:     "testutil.Contains only the permission fragment",
			output:   "without `workflows` permission",
			expected: true,
		},
		{
			name:     "regular push failure",
			output:   "error: failed to push some refs to 'https://github.com/foo/bar.git'",
			expected: false,
		},
		{
			name:     "empty output",
			output:   "",
			expected: false,
		},
		{
			name:     "unrelated rejection",
			output:   "! [remote rejected] main -> main (pre-receive hook declined)",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWorkflowPermissionError(tt.output)
			if result != tt.expected {
				t.Errorf("isWorkflowPermissionError(%q) = %v, want %v", tt.output, result, tt.expected)
			}
		})
	}
}

func TestPushBranch_WorkflowPermissionError(t *testing.T) {
	// Set up a local bare remote that rejects pushes with the GitHub workflow
	// permission message, so we can exercise the error-detection path without
	// a real GitHub connection.
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	// Write a pre-receive hook that mimics GitHub's workflow-permission rejection.
	hookContent := "#!/bin/sh\necho 'refusing to allow a GitHub App to create or update workflow `.github/workflows/test.yaml` without `workflows` permission' >&2\nexit 1\n"
	hookPath := filepath.Join(remoteDir, "hooks", "pre-receive")
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("failed to write hook: %v", err)
	}

	// Clone the bare remote into a working copy.
	workDir := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone: %v", err)
	}

	// Configure identity so commits work.
	for _, args := range [][]string{
		{"config", "--local", "user.email", "test@example.com"},
		{"config", "--local", "user.name", "Test User"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if err := c.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	// Create a commit so there is something to push.
	wfDir := filepath.Join(workDir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatalf("failed to create workflow dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "test.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "Add workflow file"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if err := c.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	t.Chdir(workDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	branch, err := GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	_, pushErr := PushBranch(ctx, branch)
	if pushErr == nil {
		t.Fatal("expected PushBranch to return an error, got nil")
	}
	if !errors.Is(pushErr, ErrWorkflowPermission) {
		t.Errorf("expected ErrWorkflowPermission, got: %v", pushErr)
	}
}

func TestPushCurrentBranch_WorkflowPermissionError(t *testing.T) {
	// Same setup as TestPushBranch_WorkflowPermissionError but exercises
	// PushCurrentBranch.
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	hookContent := "#!/bin/sh\necho 'refusing to allow a GitHub App to create or update workflow `.github/workflows/test.yaml` without `workflows` permission' >&2\nexit 1\n"
	hookPath := filepath.Join(remoteDir, "hooks", "pre-receive")
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("failed to write hook: %v", err)
	}

	workDir := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone: %v", err)
	}

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test@example.com"},
		{"config", "--local", "user.name", "Test User"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if err := c.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	wfDir := filepath.Join(workDir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatalf("failed to create workflow dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "test.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "Add workflow file"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if err := c.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}

	t.Chdir(workDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	pushErr := PushCurrentBranch(ctx)
	if pushErr == nil {
		t.Fatal("expected PushCurrentBranch to return an error, got nil")
	}
	if !errors.Is(pushErr, ErrWorkflowPermission) {
		t.Errorf("expected ErrWorkflowPermission, got: %v", pushErr)
	}
}

func TestGetCommitLog_SinceBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Get the current branch to use as base
	baseBranch, _ := GetCurrentBranch(ctx)

	// Create and checkout a new branch
	testBranch := "feature-branch"
	if err := CheckoutOrCreateBranch(ctx, testBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Create commits on the new branch
	for i := 1; i <= 2; i++ {
		testFile := filepath.Join(tempDir, "feature"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(testFile, []byte("feature content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to stage files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "Feature commit "+string(rune('0'+i)))
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create commit: %v", err)
		}
	}

	// Get commits since base branch
	commitLog, err := GetCommitLog(ctx, baseBranch, 0)
	if err != nil {
		t.Fatalf("GetCommitLog failed: %v", err)
	}

	// Should contain our feature commits
	if !testutil.Contains(commitLog, "Feature commit") {
		t.Errorf("Expected commit log to contain 'Feature commit', got: %s", commitLog)
	}

	// Should have both commits
	if !testutil.Contains(commitLog, "Feature commit 1") || !testutil.Contains(commitLog, "Feature commit 2") {
		t.Errorf("Expected both feature commits in log, got: %s", commitLog)
	}
}

func TestGetCommitLog_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	commitLog, err := GetCommitLog(ctx, "main", 0)
	if err != nil {
		t.Fatalf("GetCommitLog in dry-run failed: %v", err)
	}

	if commitLog == "" {
		t.Error("Expected dry-run commit log to be non-empty")
	}

	// Should contain dry-run format with colons
	if !testutil.Contains(commitLog, ":") {
		t.Error("Expected dry-run commit log to contain ':' separator")
	}
}

func TestGetCommitLog_EmptyRange(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Get commits since HEAD (should be empty)
	commitLog, err := GetCommitLog(ctx, "HEAD", 0)
	if err != nil {
		t.Fatalf("GetCommitLog failed: %v", err)
	}

	if commitLog != "" {
		t.Errorf("Expected empty commit log when comparing HEAD to HEAD, got: %s", commitLog)
	}
}

func TestGetDiffSince(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Get the current branch to use as base
	baseBranch, _ := GetCurrentBranch(ctx)

	// Create and checkout a new branch
	testBranch := "diff-test-branch"
	if err := CheckoutOrCreateBranch(ctx, testBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Make a change
	testFile := filepath.Join(tempDir, "changed.txt")
	if err := os.WriteFile(testFile, []byte("new content\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add changed.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	// Get diff since base branch
	diff, err := GetDiffSince(ctx, baseBranch)
	if err != nil {
		t.Fatalf("GetDiffSince failed: %v", err)
	}

	// Diff should contain the new file
	if !testutil.Contains(diff, "changed.txt") {
		t.Error("Expected diff to contain 'changed.txt'")
	}

	if !testutil.Contains(diff, "new content") {
		t.Error("Expected diff to contain 'new content'")
	}
}

func TestGetDiffSince_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	diff, err := GetDiffSince(ctx, "main")
	if err != nil {
		t.Fatalf("GetDiffSince in dry-run failed: %v", err)
	}

	if diff == "" {
		t.Error("Expected dry-run diff to be non-empty")
	}
}

func TestGetDiffSince_NoDiff(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Get diff since HEAD (should be empty)
	diff, err := GetDiffSince(ctx, "HEAD")
	if err != nil {
		t.Fatalf("GetDiffSince failed: %v", err)
	}

	if diff != "" {
		t.Errorf("Expected empty diff when comparing HEAD to HEAD, got: %s", diff)
	}
}

func TestStageFile(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Create a new file
	testFile := filepath.Join(tempDir, "newfile.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Stage the file
	err := StageFile(ctx, testFile)
	if err != nil {
		t.Fatalf("StageFile failed: %v", err)
	}

	// Verify the file is staged by checking git status
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = tempDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run git status: %v", err)
	}

	statusOutput := string(output)
	if !testutil.Contains(statusOutput, "newfile.txt") {
		t.Errorf("Expected newfile.txt to be staged, git status output: %s", statusOutput)
	}

	// Check for 'A' (added) flag
	if !testutil.Contains(statusOutput, "A") {
		t.Errorf("Expected file to be marked as Added in git status, got: %s", statusOutput)
	}
}

func TestStageFile_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	// Should not return error in dry-run mode, even for non-existent file
	err := StageFile(ctx, "/tmp/nonexistent.txt")
	if err != nil {
		t.Errorf("StageFile in dry-run should not fail: %v", err)
	}
}

func TestStageFile_NonExistent(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Try to stage a non-existent file
	err := StageFile(ctx, filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Expected error when staging non-existent file, got nil")
	}
}

func TestCommitChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Create a new file to commit
	testFile := filepath.Join(tempDir, "new-file.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Commit the changes
	err := CommitChanges(ctx)
	if err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Verify commit was created by checking log (HEAD~1 is the parent commit)
	commitLog, err := GetCommitLog(ctx, "HEAD~1", 0)
	if err != nil {
		t.Fatalf("Failed to get commit log: %v", err)
	}

	if commitLog == "" {
		t.Error("Expected at least 1 commit after CommitChanges")
	}

	// Check that the commit message mentions the file
	if !testutil.Contains(commitLog, "new-file.txt") {
		t.Logf("Commit message: %s", commitLog)
	}
}

func TestCommitChanges_NoChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Try to commit with no changes
	err := CommitChanges(ctx)
	if err == nil {
		t.Error("Expected error when committing with no changes, got nil")
	}

	expectedMsg := "no changes to commit"
	if err != nil && !testutil.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedMsg, err)
	}
}

func TestCommitChanges_MultipleFiles(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	// Create multiple files
	for i := 1; i <= 5; i++ {
		testFile := filepath.Join(tempDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Commit the changes
	err := CommitChanges(ctx)
	if err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Verify commit was created (HEAD~1 is the parent commit)
	commitLog, err := GetCommitLog(ctx, "HEAD~1", 0)
	if err != nil {
		t.Fatalf("Failed to get commit log: %v", err)
	}

	if commitLog == "" {
		t.Error("Expected at least 1 commit after CommitChanges")
	}

	// For multiple files, should have a summary message
	if commitLog != "" {
		t.Logf("Commit message for multiple files: %s", commitLog)
	}
}

func TestCommitChanges_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	// Should not error in dry-run mode
	err := CommitChanges(ctx)
	if err != nil {
		t.Errorf("CommitChanges in dry-run should not error, got: %v", err)
	}
}

func TestFindRepoRoot(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	root, err := FindRepoRoot(ctx)
	if err != nil {
		t.Fatalf("FindRepoRoot failed: %v", err)
	}

	if root != tempDir {
		t.Errorf("Expected repo root to be %q, got %q", tempDir, root)
	}
}

func TestFindRepoRoot_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ctx := testutil.NewContext()

	root, err := FindRepoRoot(ctx)
	if err != nil {
		t.Fatalf("FindRepoRoot in dry-run failed: %v", err)
	}

	if root != tempDir {
		t.Errorf("Expected dry-run to return cwd %q, got %q", tempDir, root)
	}
}

func TestFindRepoRoot_NotARepo(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))

	_, err := FindRepoRoot(ctx)
	if err == nil {
		t.Error("Expected error when FindRepoRoot is called outside a git repository")
	}
}

func TestIsBranchSyncedWithRemote_Synced(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)

	branchName := "synced-branch"
	for _, args := range [][]string{
		{"checkout", "-b", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(workDir, "test.txt"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
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

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	if err := IsBranchSyncedWithRemote(ctx, branchName); err != nil {
		t.Errorf("IsBranchSyncedWithRemote failed for synced branch: %v", err)
	}
}

func TestIsBranchSyncedWithRemote_Behind(t *testing.T) {
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	workDir1 := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir1)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone: %v", err)
	}

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test@example.com"},
		{"config", "--local", "user.name", "Test User"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir1
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	branchName := "behind-branch"
	for _, args := range [][]string{
		{"checkout", "-b", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir1
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(workDir1, "test.txt"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "add test file"},
		{"push", "origin", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir1
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	workDir2 := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir2)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone: %v", err)
	}

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test2@example.com"},
		{"config", "--local", "user.name", "Test User 2"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir2
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	cmd = exec.Command("git", "checkout", branchName)
	cmd.Dir = workDir2
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to checkout branch: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workDir2, "test2.txt"), []byte("test2\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "add test2 file"},
		{"push", "origin", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir2
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	t.Chdir(workDir1)

	cmd = exec.Command("git", "fetch", "origin")
	cmd.Dir = workDir1
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	err := IsBranchSyncedWithRemote(ctx, branchName)
	if err == nil {
		t.Error("Expected error when local branch is behind remote")
	}
}

func TestIsBranchSyncedWithRemote_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	if err := IsBranchSyncedWithRemote(ctx, "any-branch"); err != nil {
		t.Errorf("IsBranchSyncedWithRemote in dry-run failed: %v", err)
	}
}

func TestPullRebase_WithNewCommits(t *testing.T) {
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	workDir1 := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir1)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone: %v", err)
	}

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test@example.com"},
		{"config", "--local", "user.name", "Test User"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir1
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	branchName := "pull-test-branch"
	for _, args := range [][]string{
		{"checkout", "-b", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir1
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(workDir1, "test.txt"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "add test file"},
		{"push", "origin", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir1
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	workDir2 := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir2)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone: %v", err)
	}

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test2@example.com"},
		{"config", "--local", "user.name", "Test User 2"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir2
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	cmd = exec.Command("git", "checkout", branchName)
	cmd.Dir = workDir2
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to checkout branch: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workDir2, "test2.txt"), []byte("test2\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "add test2 file"},
		{"push", "origin", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir2
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	t.Chdir(workDir1)

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	if err := PullRebase(ctx); err != nil {
		t.Fatalf("PullRebase failed: %v", err)
	}
}

func TestPullRebase_DryRun(t *testing.T) {
	ctx := testutil.NewContext()

	if err := PullRebase(ctx); err != nil {
		t.Errorf("PullRebase in dry-run failed: %v", err)
	}
}

func TestDeleteFile(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	testFile := filepath.Join(tempDir, "to-delete.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command("git", "add", "to-delete.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "add file to delete")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	if err := DeleteFile(ctx, testFile); err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	if _, err := os.Stat(testFile); err == nil {
		t.Error("Expected file to be deleted")
	}

	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = tempDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run git status: %v", err)
	}

	statusOutput := string(output)
	if !testutil.Contains(statusOutput, "D") {
		t.Errorf("Expected deletion to be staged (D), got: %s", statusOutput)
	}
}

func TestDeleteFile_NonExistent(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	err := DeleteFile(ctx, filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Expected error when deleting non-existent file")
	}
}

func TestDeleteFile_DryRun(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	testFile := filepath.Join(tempDir, "to-delete-dryrun.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := testutil.NewContext()
	if err := DeleteFile(ctx, testFile); err != nil {
		t.Errorf("DeleteFile in dry-run failed: %v", err)
	}

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should NOT be deleted in dry-run mode")
	}
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

	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to create README: %v", err)
	}
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

	branchName := "existing-remote-branch"
	for _, args := range [][]string{
		{"checkout", "-b", branchName},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(workDir, "test.txt"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
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

	workDir2 := t.TempDir()
	t.Logf("About to clone from %s to %s", remoteDir, workDir2)
	cwd, _ := os.Getwd()
	t.Logf("Current working directory: %s", cwd)
	cmd = exec.Command("git", "clone", remoteDir, workDir2)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to clone: %v\n%s", err, out)
	}

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test2@example.com"},
		{"config", "--local", "user.name", "Test User 2"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir2
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	t.Chdir(workDir2)

	ctx := testutil.NewContext(testutil.WithDryRun(false))
	if err := CheckoutOrCreateBranch(ctx, branchName); err != nil {
		t.Fatalf("CheckoutOrCreateBranch failed: %v", err)
	}

	currentBranch, err := GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != branchName {
		t.Errorf("Expected current branch to be %q, got %q", branchName, currentBranch)
	}

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = workDir2
	localHash, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get local hash: %v", err)
	}

	cmd = exec.Command("git", "rev-parse", "origin/"+branchName)
	cmd.Dir = workDir2
	remoteHash, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get remote hash: %v", err)
	}

	if string(localHash) != string(remoteHash) {
		t.Error("Local branch should be at same commit as remote branch after checkout")
	}
}

func TestCategorizeFile(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"path/to/file.go", "path"},
		{"another/path/file.ts", "another"},
		{"other/file.py", "other"},
		{"file.go", "go"},
		{"script.py", "py"},
		{"noextension", "root"},
		{"path/to/multiple.dots.tar.gz", "path"},
		{"just/a/folder/", "just"},
	}

	for _, tt := range tests {
		result := categorizeFile(tt.file)
		if result != tt.expected {
			t.Errorf("categorizeFile(%q) = %q, want %q", tt.file, result, tt.expected)
		}
	}
}

func TestCategorizeFiles(t *testing.T) {
	files := []string{
		"path/to/file.go",
		"path/to/another.ts",
		"other/file.py",
		"file.go",
		"noextension",
	}

	categories := categorizeFiles(files)

	expected := map[string]int{
		"path":  2,
		"other": 1,
		"go":    1,
		"root":  1,
	}

	for category, count := range expected {
		if categories[category] != count {
			t.Errorf("categorizeFiles[%q] = %d, want %d", category, categories[category], count)
		}
	}
}

func TestBuildCommitMessage(t *testing.T) {
	tests := []struct {
		files     []string
		fileCount int
		expected  string
	}{
		{[]string{"file.go"}, 1, "Update file.go"},
		{[]string{"file1.go", "file2.ts"}, 2, "Update file1.go, file2.ts"},
		{[]string{"a.go", "b.ts", "c.py"}, 3, "Update a.go, b.ts, c.py"},
		{[]string{"a.go", "b.ts", "c.py", "d.rb"}, 4, "Update 4 files across project"},
	}

	for _, tt := range tests {
		result := buildCommitMessage(tt.files, tt.fileCount)
		if result != tt.expected {
			t.Errorf("buildCommitMessage(%v, %d) = %q, want %q", tt.files, tt.fileCount, result, tt.expected)
		}
	}
}

func TestBuildCommitMessage_SingleCategory(t *testing.T) {
	files := []string{"path/to/file1.go", "path/to/file2.go", "path/to/file3.go"}
	result := buildCommitMessage(files, 4)
	expected := "Update path files (4 files)"
	if result != expected {
		t.Errorf("buildCommitMessage with single category = %q, want %q", result, expected)
	}
}

func TestSummarizeCommitMessage(t *testing.T) {
	singleCategory := []string{"dir/file1.go", "dir/file2.go"}
	result := summarizeCommitMessage(singleCategory, 2)
	if result != "Update dir files (2 files)" {
		t.Errorf("summarizeCommitMessage(single category) = %q, want %q", result, "Update dir files (2 files)")
	}

	multiCategory := []string{"dir1/file.go", "dir2/file.go"}
	result = summarizeCommitMessage(multiCategory, 2)
	if result != "Update 2 files across project" {
		t.Errorf("summarizeCommitMessage(multi category) = %q, want %q", result, "Update 2 files across project")
	}
}
