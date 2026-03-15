package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGitRepository(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Should be true inside a git repository
	assert.True(t, IsGitRepository(), "Expected IsGitRepository to return true inside a git repo")
}

func TestIsGitRepository_NotRepo(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	// Should be false outside a git repository
	assert.False(t, IsGitRepository(), "Expected IsGitRepository to return false outside a git repo")
}

func TestIsDetachedHead(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Should not be detached on a normal branch
	isDetached, err := IsDetachedHead()
	require.NoError(t, err, "IsDetachedHead failed")

	assert.False(t, isDetached, "Expected IsDetachedHead to return false on a branch")
}

func TestFindRepoRoot(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	root, err := FindRepoRoot()
	require.NoError(t, err, "FindRepoRoot failed")

	assert.Equal(t, tempDir, root)
}

func TestFindRepoRoot_NotARepo(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	_, err := FindRepoRoot()
	require.Error(t, err, "Expected error when FindRepoRoot is called outside a git repository")
}

func TestRevParse(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	// Test --show-toplevel (should match FindRepoRoot logic)
	root, err := RevParse("--show-toplevel")
	require.NoError(t, err)
	assert.Equal(t, tempDir, root)

	// Test --abbrev-ref HEAD (should match GetCurrentBranch logic)
	branch, err := RevParse("--abbrev-ref", "HEAD")
	require.NoError(t, err)
	assert.True(t, branch == "master" || branch == "main")
}
