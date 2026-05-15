package git

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepoRoot(t *testing.T) {
	t.Run("returns repo root when inside a git repo", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		origWd, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(origWd) })
		os.Chdir(tempDir)

		root, err := RepoRoot()
		require.NoError(t, err)
		require.Equal(t, tempDir, root)
	})

	t.Run("returns error when not inside a git repo", func(t *testing.T) {
		tempDir := t.TempDir()
		origWd, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(origWd) })
		os.Chdir(tempDir)

		_, err := RepoRoot()
		require.Error(t, err)
	})
}