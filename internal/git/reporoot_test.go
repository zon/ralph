package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepoRootOrCwd(t *testing.T) {
	t.Run("returns repo root when inside a git repo", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		root := RepoRootOrCwd()
		require.Equal(t, tempDir, root)
	})

	t.Run("returns cwd when not inside a git repo", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Chdir(tempDir)

		root := RepoRootOrCwd()
		require.Equal(t, tempDir, root)
	})
}
