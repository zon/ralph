package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
