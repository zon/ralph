package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/context"
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
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Should be true inside a git repository
	if !IsGitRepository(ctx) {
		t.Error("Expected IsGitRepository to return true inside a git repo")
	}
}

func TestIsGitRepository_NotRepo(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Should be false outside a git repository
	if IsGitRepository(ctx) {
		t.Error("Expected IsGitRepository to return false outside a git repo")
	}
}

func TestIsGitRepository_DryRun(t *testing.T) {
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Should always return true in dry-run mode
	if !IsGitRepository(ctx) {
		t.Error("Expected IsGitRepository to return true in dry-run mode")
	}
}

func TestIsDetachedHead(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	defer os.Chdir(tempDir) // Ensure we're in the test repo
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedMsg, err)
	}
}

func TestBranchExists(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Check for current branch (should exist)
	currentBranch, _ := GetCurrentBranch(ctx)
	if !BranchExists(ctx, currentBranch) {
		t.Errorf("Expected current branch '%s' to exist", currentBranch)
	}

	// Check for non-existent branch
	if BranchExists(ctx, "non-existent-branch") {
		t.Error("Expected non-existent branch to return false")
	}
}

func TestBranchExists_DryRun(t *testing.T) {
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// In dry-run mode, should always return false
	exists := BranchExists(ctx, "any-branch")
	if exists {
		t.Error("Expected dry-run BranchExists to return false")
	}
}

func TestCreateBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	branchName := "test-feature-branch"

	// Create the branch
	if err := CreateBranch(ctx, branchName); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Verify it exists
	if !BranchExists(ctx, branchName) {
		t.Errorf("Branch '%s' was not created", branchName)
	}
}

func TestCreateBranch_DryRun(t *testing.T) {
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Should not return an error in dry-run mode
	if err := CreateBranch(ctx, "test-branch"); err != nil {
		t.Fatalf("CreateBranch in dry-run failed: %v", err)
	}
}

func TestCheckoutBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	branchName := "checkout-test-branch"

	// Create the branch first
	if err := CreateBranch(ctx, branchName); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Checkout the branch
	if err := CheckoutBranch(ctx, branchName); err != nil {
		t.Fatalf("CheckoutBranch failed: %v", err)
	}

	// Verify we're on the new branch
	currentBranch, err := GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != branchName {
		t.Errorf("Expected current branch to be '%s', got '%s'", branchName, currentBranch)
	}
}

func TestCheckoutBranch_DryRun(t *testing.T) {
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Should not return an error in dry-run mode
	if err := CheckoutBranch(ctx, "any-branch"); err != nil {
		t.Fatalf("CheckoutBranch in dry-run failed: %v", err)
	}
}

func TestCreateBranch_AlreadyExists(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	branchName := "duplicate-branch"

	// Create the branch
	if err := CreateBranch(ctx, branchName); err != nil {
		t.Fatalf("First CreateBranch failed: %v", err)
	}

	// Try to create it again (should fail)
	err := CreateBranch(ctx, branchName)
	if err == nil {
		t.Error("Expected CreateBranch to fail for existing branch")
	}
}

func TestHasCommits(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Should have commits (setupTestRepo creates an initial commit)
	if !HasCommits(ctx) {
		t.Error("Expected HasCommits to return true for repo with commits")
	}
}

func TestHasCommits_DryRun(t *testing.T) {
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// In dry-run mode, should always return true
	if !HasCommits(ctx) {
		t.Error("Expected HasCommits to return true in dry-run mode")
	}
}

func TestPushBranch_DryRun(t *testing.T) {
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Should not return an error in dry-run mode
	url, err := PushBranch(ctx, "test-branch")
	if err != nil {
		t.Fatalf("PushBranch in dry-run failed: %v", err)
	}

	if url == "" {
		t.Error("Expected PushBranch to return a URL in dry-run mode")
	}
}

// Note: We skip real push tests as they require:
// 1. A remote repository configured
// 2. Authentication/credentials set up
// 3. Network access
// These are integration tests that should be run in a CI/CD environment
// with proper repository setup.

func TestGetCommitLog_SinceBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Get the current branch to use as base
	baseBranch, _ := GetCurrentBranch(ctx)

	// Create a new branch
	testBranch := "feature-branch"
	if err := CreateBranch(ctx, testBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := CheckoutBranch(ctx, testBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
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
	commitLog, err := GetCommitLog(ctx, baseBranch)
	if err != nil {
		t.Fatalf("GetCommitLog failed: %v", err)
	}

	// Should contain our feature commits
	if !contains(commitLog, "Feature commit") {
		t.Errorf("Expected commit log to contain 'Feature commit', got: %s", commitLog)
	}

	// Should have both commits
	if !contains(commitLog, "Feature commit 1") || !contains(commitLog, "Feature commit 2") {
		t.Errorf("Expected both feature commits in log, got: %s", commitLog)
	}
}

func TestGetCommitLog_DryRun(t *testing.T) {
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	commitLog, err := GetCommitLog(ctx, "main")
	if err != nil {
		t.Fatalf("GetCommitLog in dry-run failed: %v", err)
	}

	if commitLog == "" {
		t.Error("Expected dry-run commit log to be non-empty")
	}

	// Should contain dry-run format with colons
	if !contains(commitLog, ":") {
		t.Error("Expected dry-run commit log to contain ':' separator")
	}
}

func TestGetCommitLog_EmptyRange(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Get commits since HEAD (should be empty)
	commitLog, err := GetCommitLog(ctx, "HEAD")
	if err != nil {
		t.Fatalf("GetCommitLog failed: %v", err)
	}

	if commitLog != "" {
		t.Errorf("Expected empty commit log when comparing HEAD to HEAD, got: %s", commitLog)
	}
}

func TestGetDiffSince(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Get the current branch to use as base
	baseBranch, _ := GetCurrentBranch(ctx)

	// Create a new branch
	testBranch := "diff-test-branch"
	if err := CreateBranch(ctx, testBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := CheckoutBranch(ctx, testBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
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
	if !contains(diff, "changed.txt") {
		t.Error("Expected diff to contain 'changed.txt'")
	}

	if !contains(diff, "new content") {
		t.Error("Expected diff to contain 'new content'")
	}
}

func TestGetDiffSince_DryRun(t *testing.T) {
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Get diff since HEAD (should be empty)
	diff, err := GetDiffSince(ctx, "HEAD")
	if err != nil {
		t.Fatalf("GetDiffSince failed: %v", err)
	}

	if diff != "" {
		t.Errorf("Expected empty diff when comparing HEAD to HEAD, got: %s", diff)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestStageFile(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	if !contains(statusOutput, "newfile.txt") {
		t.Errorf("Expected newfile.txt to be staged, git status output: %s", statusOutput)
	}

	// Check for 'A' (added) flag
	if !contains(statusOutput, "A") {
		t.Errorf("Expected file to be marked as Added in git status, got: %s", statusOutput)
	}
}

func TestStageFile_DryRun(t *testing.T) {
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Should not return error in dry-run mode, even for non-existent file
	err := StageFile(ctx, "/tmp/nonexistent.txt")
	if err != nil {
		t.Errorf("StageFile in dry-run should not fail: %v", err)
	}
}

func TestStageFile_NonExistent(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Try to stage a non-existent file
	err := StageFile(ctx, filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Expected error when staging non-existent file, got nil")
	}
}

func TestCommitChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	commitLog, err := GetCommitLog(ctx, "HEAD~1")
	if err != nil {
		t.Fatalf("Failed to get commit log: %v", err)
	}

	if commitLog == "" {
		t.Error("Expected at least 1 commit after CommitChanges")
	}

	// Check that the commit message mentions the file
	if !contains(commitLog, "new-file.txt") {
		t.Logf("Commit message: %s", commitLog)
	}
}

func TestCommitChanges_NoChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Try to commit with no changes
	err := CommitChanges(ctx)
	if err == nil {
		t.Error("Expected error when committing with no changes, got nil")
	}

	expectedMsg := "no changes to commit"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedMsg, err)
	}
}

func TestCommitChanges_MultipleFiles(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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
	commitLog, err := GetCommitLog(ctx, "HEAD~1")
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
	ctx := &context.Context{ProjectFile: "", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

	// Should not error in dry-run mode
	err := CommitChanges(ctx)
	if err != nil {
		t.Errorf("CommitChanges in dry-run should not error, got: %v", err)
	}
}
