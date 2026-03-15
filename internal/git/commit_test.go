package git

import (
	"os"
	"path/filepath"
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
	err := CommitChanges()
	require.NoError(t, err, "CommitChanges failed")

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
	err := CommitChanges()
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

	if err := DeleteFile("to-delete.txt"); err != nil {
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
