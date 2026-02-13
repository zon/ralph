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

func TestGetCurrentBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	defer os.Chdir(tempDir) // Ensure we're in the test repo
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := context.NewContext(false, false, false, false)

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
	ctx := context.NewContext(true, false, false, false)

	branch, err := GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("GetCurrentBranch in dry-run failed: %v", err)
	}

	if branch != "dry-run-branch" {
		t.Errorf("Expected dry-run branch to be 'dry-run-branch', got '%s'", branch)
	}
}

func TestBranchExists(t *testing.T) {
	tempDir := setupTestRepo(t)
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	ctx := context.NewContext(false, false, false, false)

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
	ctx := context.NewContext(true, false, false, false)

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

	ctx := context.NewContext(false, false, false, false)

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
	ctx := context.NewContext(true, false, false, false)

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

	ctx := context.NewContext(false, false, false, false)

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
	ctx := context.NewContext(true, false, false, false)

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

	ctx := context.NewContext(false, false, false, false)

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

	ctx := context.NewContext(false, false, false, false)

	// Should have commits (setupTestRepo creates an initial commit)
	if !HasCommits(ctx) {
		t.Error("Expected HasCommits to return true for repo with commits")
	}
}

func TestHasCommits_DryRun(t *testing.T) {
	ctx := context.NewContext(true, false, false, false)

	// In dry-run mode, should always return true
	if !HasCommits(ctx) {
		t.Error("Expected HasCommits to return true in dry-run mode")
	}
}

func TestPushBranch_DryRun(t *testing.T) {
	ctx := context.NewContext(true, false, false, false)

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
