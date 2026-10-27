package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientHasCommitsAhead(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	client := NewClient(nil)
	base, err := client.CurrentBranch()
	require.NoError(t, err)

	require.NoError(t, CheckoutOrCreateBranch("ahead-branch"))
	ahead, err := client.HasCommitsAhead(base)
	require.NoError(t, err)
	require.False(t, ahead, "a branch with no commits of its own is not ahead")

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "b.txt"), []byte("b"), 0644))
	require.NoError(t, StageFile("b.txt"))
	require.NoError(t, Commit("feat: b"))

	ahead, err = client.HasCommitsAhead(base)
	require.NoError(t, err)
	require.True(t, ahead, "a branch with its own commit is ahead of the base")
}

func TestClientHeadCommit(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	client := NewClient(nil)
	before, err := client.HeadCommit()
	require.NoError(t, err)
	require.NotEmpty(t, before)

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "tip.txt"), []byte("tip"), 0644))
	require.NoError(t, StageFile("tip.txt"))
	require.NoError(t, Commit("feat: advance the tip"))

	after, err := client.HeadCommit()
	require.NoError(t, err)
	require.NotEqual(t, before, after, "a new commit advances the branch tip")
}

func TestClientCheckoutBranch(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	client := NewClient(nil)
	base, err := client.CurrentBranch()
	require.NoError(t, err)
	require.NoError(t, CreateBranch("restore-branch"))

	require.NoError(t, client.CheckoutBranch(base))
	current, err := client.CurrentBranch()
	require.NoError(t, err)
	require.Equal(t, base, current)
}
