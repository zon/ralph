package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageFile(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Create a new file
	testFile := filepath.Join(tempDir, "newfile.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Stage the file
	err := StageFile("newfile.txt")
	require.NoError(t, err, "StageFile failed")

	// Verify the file is staged by checking git status
	status, err := RevParse("--verify", ":newfile.txt")
	require.NoError(t, err, "Failed to verify staged file")
	assert.NotEmpty(t, status)
}

func TestStageFile_NonExistent(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Try to stage a non-existent file
	err := StageFile("nonexistent.txt")
	require.Error(t, err, "Expected error when staging non-existent file")
}

func TestCommitChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Create a new file to commit
	testFile := filepath.Join(tempDir, "new-file.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Commit the changes
	err := commitChanges()
	require.NoError(t, err, "commitChanges failed")

	// Verify commit was created by checking log
	commitLog, err := GetCommitLog("HEAD~1", 1)
	require.NoError(t, err, "Failed to get commit log")

	assert.NotEmpty(t, commitLog, "Expected at least 1 commit after CommitChanges")
	assert.Contains(t, commitLog, "new-file.txt")
}

func TestCommitChanges_NoChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Try to commit with no changes
	err := commitChanges()
	require.Error(t, err, "Expected error when committing with no changes")
	assert.Contains(t, err.Error(), "no changes to commit")
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
	}

	for _, tt := range tests {
		result := categorizeFile(tt.file)
		assert.Equal(t, tt.expected, result)
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
		assert.Equal(t, count, categories[category])
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
		assert.Equal(t, tt.expected, result)
	}
}

func TestDeleteFile(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	testFile := filepath.Join(tempDir, "to-delete.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := StageFile("to-delete.txt")
	require.NoError(t, err)
	err = Commit("add file to delete")
	require.NoError(t, err)

	if err := deleteFile("to-delete.txt"); err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err), "Expected file to be deleted from filesystem")
}

func TestHasUncommittedChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	assert.False(t, HasUncommittedChanges())

	// Unstaged change
	if err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("modified\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}
	assert.True(t, HasUncommittedChanges())
}

func TestBranchLogContainsPrefix(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Get the base commit (initial commit on main/master)
	base, err := GetCurrentBranch()
	require.NoError(t, err)

	// Create a feature branch
	err = CheckoutOrCreateBranch("review-2026-03-25")
	require.NoError(t, err)

	// Add a commit with a known prefix
	err = os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)
	err = StageFile("file1.txt")
	require.NoError(t, err)
	err = Commit("internal-git-0 Found missing error handling in commit flow")
	require.NoError(t, err)

	// Add another commit with a different prefix
	err = os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)
	err = StageFile("file2.txt")
	require.NoError(t, err)
	err = Commit("internal-api-1 API docs need updating")
	require.NoError(t, err)

	tests := []struct {
		name     string
		prefix   string
		expected bool
	}{
		{
			name:     "prefix present in history",
			prefix:   "internal-git-0",
			expected: true,
		},
		{
			name:     "different prefix present",
			prefix:   "internal-api-1",
			expected: true,
		},
		{
			name:     "prefix not in history",
			prefix:   "internal-git-1",
			expected: false,
		},
		{
			name:     "prefix with different component",
			prefix:   "cmd-0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := BranchLogContainsPrefix(base, "review-2026-03-25", tt.prefix)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, found)
		})
	}
}

func TestBranchLogContainsPrefix_NonExistentBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)

	// Non-existent branch should return false without error
	found, err := BranchLogContainsPrefix(base, "non-existent-branch", "some-prefix")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestBranchLogContainsPrefix_EmptyBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)

	// Branch with no commits ahead of base should return false
	err = CheckoutOrCreateBranch("empty-branch")
	require.NoError(t, err)

	found, err := BranchLogContainsPrefix(base, "empty-branch", "some-prefix")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestSwitchToBranchIfNeeded(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	branchName := "review-branch"

	err := SwitchToBranchIfNeeded(nil, branchName)
	require.NoError(t, err, "SwitchToBranchIfNeeded failed")

	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")
	assert.Equal(t, branchName, currentBranch)
}

func TestSwitchToBranchIfNeeded_AlreadyOnBranch(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	branchName, err := GetCurrentBranch()
	require.NoError(t, err, "GetCurrentBranch failed")

	err = SwitchToBranchIfNeeded(nil, branchName)
	require.NoError(t, err, "SwitchToBranchIfNeeded should succeed when already on branch")
}

func TestCommitFileAndPush(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	branchName := "review-branch"
	filePath := "test.txt"
	commitMsg := "Add test file"

	err := os.WriteFile(filepath.Join(workDir, filePath), []byte("test content"), 0644)
	require.NoError(t, err)

	err = CommitFileAndPush(nil, filePath, branchName, commitMsg)
	require.NoError(t, err, "CommitFileAndPush failed")

	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, branchName, currentBranch)

	hasChanges := HasUncommittedChanges()
	assert.False(t, hasChanges, "Should not have uncommitted changes after commit")
}

func TestCommitAllAndPush(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	branchName := "review-all-branch"
	commitMsg := "Add multiple files"

	err := os.MkdirAll(filepath.Join(workDir, "projects"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(workDir, "projects", "test.yaml"), []byte("name: test\n"), 0644)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(workDir, "docs"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(workDir, "docs", "notes.md"), []byte("notes\n"), 0644)
	require.NoError(t, err)

	err = CommitAllAndPush(nil, branchName, commitMsg)
	require.NoError(t, err, "CommitAllAndPush failed")

	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, branchName, currentBranch)

	hasChanges := HasUncommittedChanges()
	assert.False(t, hasChanges, "Should not have uncommitted changes after commit")
}

func TestPerformCommit_WithStagedChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	require.NoError(t, StageFile("test.txt"))

	err := performCommit("Add test file")
	require.NoError(t, err, "performCommit failed")

	hasStaged := HasStagedChanges()
	assert.False(t, hasStaged, "Should have no staged changes after commit")
}

func TestPerformCommit_EmptyMessage(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	require.NoError(t, StageFile("test.txt"))

	err := performCommit("")
	require.Error(t, err, "performCommit should fail with empty message")
	assert.Contains(t, err.Error(), "empty commit message")
}

func TestPerformCommit_NoStagedChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	err := performCommit("Some commit message")
	require.Error(t, err, "performCommit should fail with no staged changes")
	assert.True(t, errors.Is(err, ErrNoChanges), "Expected ErrNoChanges, got: %v", err)
}

func TestCommitChanges_WithStagedChanges(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	testFile := filepath.Join(workDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := CommitChanges(false, "", "", "Add test file")
	require.NoError(t, err, "CommitChanges failed")

	hasChanges := HasUncommittedChanges()
	assert.False(t, hasChanges, "Should have no uncommitted changes after CommitChanges")

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git log failed")
	msg := strings.TrimSpace(string(out))
	assert.Equal(t, "Add test file", msg)
}

func TestCommitChanges_NoStagedChanges(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	err := CommitChanges(false, "", "", "Add test file")
	require.Error(t, err, "CommitChanges should fail with no staged changes")
	assert.True(t, errors.Is(err, ErrNoChanges), "Expected ErrNoChanges, got: %v", err)
}
