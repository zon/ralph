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
	err := os.WriteFile(testFile, []byte("new content"), 0644)
	require.NoError(t, err)

	// Stage the file
	err = StageFile("newfile.txt")
	require.NoError(t, err, "StageFile failed")

	// Verify the file is staged by checking git status
	status, err := runGit("rev-parse", "--verify", ":newfile.txt")
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

func TestHasUncommittedChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	assert.False(t, HasUncommittedChanges())

	// Unstaged change
	err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("modified\n"), 0644)
	require.NoError(t, err)
	assert.True(t, HasUncommittedChanges())
}

func TestPerformCommit_WithStagedChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	require.NoError(t, StageFile("test.txt"))

	err = performCommit("Add test file", false)
	require.NoError(t, err, "performCommit failed")

	hasStaged := HasStagedChanges()
	assert.False(t, hasStaged, "Should have no staged changes after commit")
}

func TestPerformCommit_EmptyMessage(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	require.NoError(t, StageFile("test.txt"))

	err = performCommit("", false)
	require.Error(t, err, "performCommit should fail with empty message")
	assert.Contains(t, err.Error(), "empty commit message")
}

func TestPerformCommit_NoStagedChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	err := performCommit("Some commit message", false)
	require.Error(t, err, "performCommit should fail with no staged changes")
	assert.True(t, errors.Is(err, ErrNoChanges), "Expected ErrNoChanges, got: %v", err)
}

func TestPerformCommit_AllowEmptyCreatesEmptyCommitWhenNoChanges(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	message := "feat: no code needed\n\ncsv-export-1"
	err := performCommit(message, true)
	require.NoError(t, err, "performCommit with allowEmpty should succeed with no staged changes")

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, message, strings.TrimSpace(string(out)), "the empty commit carries the message verbatim")
	assertTreesEqual(t, tempDir, "HEAD", "HEAD^", "the empty commit carries no file changes")
}

func TestCommitEmpty(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	err := CommitEmpty("chore: empty commit")
	require.NoError(t, err)

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, "chore: empty commit", strings.TrimSpace(string(out)))
	assertTreesEqual(t, tempDir, "HEAD", "HEAD^", "the empty commit carries no file changes")
}

func TestCommitChanges_WithStagedChanges(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	testFile := filepath.Join(workDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = CommitChanges(false, "", "", "Add test file")
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

func TestCommitProjectRemoval(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	projectFile := filepath.Join(tempDir, "projects", "test-project.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(projectFile), 0755))
	require.NoError(t, os.WriteFile(projectFile, []byte("slug: test-project\n"), 0644))
	require.NoError(t, StageFile("projects/test-project.yaml"))
	require.NoError(t, Commit("add project file"))

	require.NoError(t, os.Remove(projectFile))
	require.NoError(t, StageFile(projectFile))

	err := CommitProjectRemoval("projects/test-project.yaml")
	require.NoError(t, err, "CommitProjectRemoval failed")

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git log failed")
	msg := strings.TrimSpace(string(out))
	assert.Equal(t, "chore: clean up completed project projects/test-project.yaml", msg)
}

func TestCommitChanges_NoStagedChanges(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	err := CommitChanges(false, "", "", "Add test file")
	require.Error(t, err, "CommitChanges should fail with no staged changes")
	assert.True(t, errors.Is(err, ErrNoChanges), "Expected ErrNoChanges, got: %v", err)
}

func TestCommitChangesAllowEmpty_CreatesEmptyCommitWhenNoChanges(t *testing.T) {
	workDir, _ := setupBareRemoteRepo(t)
	t.Chdir(workDir)

	message := "feat: no code needed\n\ncsv-export-1"
	err := CommitChangesAllowEmpty(false, "", "", message)
	require.NoError(t, err, "CommitChangesAllowEmpty should succeed with no staged changes")

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, message, strings.TrimSpace(string(out)), "the empty commit carries the report verbatim")
	assertTreesEqual(t, workDir, "HEAD", "HEAD^", "the empty commit carries no file changes")
	assert.False(t, HasUncommittedChanges(), "the working tree stays clean after the empty commit")
}

func TestCommitWorkingTree(t *testing.T) {
	t.Run("is a no-op on a clean tree", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		require.NoError(t, CommitWorkingTree("chore: no-op"))
		require.False(t, HasUncommittedChanges())
	})

	t.Run("commits uncommitted changes", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "new.txt"), []byte("x\n"), 0644))
		require.NoError(t, CommitWorkingTree("chore: sweep"))

		require.False(t, HasUncommittedChanges())
		cmd := exec.Command("git", "log", "-1", "--format=%B")
		cmd.Dir = tempDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git log failed")
		assert.Equal(t, "chore: sweep", strings.TrimSpace(string(out)))
	})
}

func assertTreesEqual(t *testing.T, dir, refA, refB string, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Equal(t, revParseTree(t, dir, refA), revParseTree(t, dir, refB), msgAndArgs...)
}

func revParseTree(t *testing.T, dir, ref string) string {
	t.Helper()
	c := exec.Command("git", "rev-parse", ref+"^{tree}")
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git rev-parse %s^{tree}", ref)
	return strings.TrimSpace(string(out))
}
