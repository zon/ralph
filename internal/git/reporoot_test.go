package git

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepoRootOrCwd(t *testing.T) {
	t.Run("returns repo root when inside a git repo", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		origWd, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(origWd) })
		os.Chdir(tempDir)

		root := RepoRootOrCwd()
		require.Equal(t, tempDir, root)
	})

	t.Run("returns cwd when not inside a git repo", func(t *testing.T) {
		tempDir := t.TempDir()
		origWd, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(origWd) })
		os.Chdir(tempDir)

		cwd, _ := os.Getwd()
		root := RepoRootOrCwd()
		require.Equal(t, cwd, root)
	})
}