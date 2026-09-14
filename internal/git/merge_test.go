package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)

	featureBranch := "feature-branch"
	err = CheckoutOrCreateBranch(featureBranch)
	require.NoError(t, err)

	featureFile := "feature.txt"
	err = os.WriteFile(filepath.Join(tempDir, featureFile), []byte("feature content"), 0644)
	require.NoError(t, err)
	err = StageFile(featureFile)
	require.NoError(t, err)
	err = Commit("feature commit")
	require.NoError(t, err)

	_, err = runGit("checkout", base)
	require.NoError(t, err)

	baseFile := "base.txt"
	err = os.WriteFile(filepath.Join(tempDir, baseFile), []byte("base content"), 0644)
	require.NoError(t, err)
	err = StageFile(baseFile)
	require.NoError(t, err)
	err = Commit("base commit")
	require.NoError(t, err)

	err = Merge(featureBranch)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(tempDir, featureFile))
	assert.NoError(t, err, "merged file should exist in working tree")
}

func TestMerge_Conflict(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)

	featureBranch := "conflict-feature"
	err = CheckoutOrCreateBranch(featureBranch)
	require.NoError(t, err)

	conflictFile := "conflict.txt"
	err = os.WriteFile(filepath.Join(tempDir, conflictFile), []byte("feature content"), 0644)
	require.NoError(t, err)
	err = StageFile(conflictFile)
	require.NoError(t, err)
	err = Commit("feature conflict commit")
	require.NoError(t, err)

	_, err = runGit("checkout", base)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, conflictFile), []byte("base content"), 0644)
	require.NoError(t, err)
	err = StageFile(conflictFile)
	require.NoError(t, err)
	err = Commit("base conflict commit")
	require.NoError(t, err)

	err = Merge(featureBranch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "merge failed")
}

// TestNeedsMerge covers the check synchronization uses to skip a merge when the
// base branch tip is already contained in the working branch.
func TestNeedsMerge(t *testing.T) {
	t.Run("identical base tip needs no merge", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		base, err := GetCurrentBranch()
		require.NoError(t, err)

		needsMerge, err := NeedsMerge(base)
		require.NoError(t, err)
		assert.False(t, needsMerge, "the working branch is the base branch, so no merge is needed")
	})

	t.Run("contained base tip needs no merge", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		base, err := GetCurrentBranch()
		require.NoError(t, err)

		require.NoError(t, CreateBranch("feature"))
		writeAndCommit(t, tempDir, "feature.txt", "feature commit")

		needsMerge, err := NeedsMerge(base)
		require.NoError(t, err)
		assert.False(t, needsMerge, "the base branch tip is an ancestor of the working branch")
	})

	t.Run("advanced base needs a merge", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		base, err := GetCurrentBranch()
		require.NoError(t, err)

		require.NoError(t, CreateBranch("feature"))
		require.NoError(t, CheckoutBranch(base))
		writeAndCommit(t, tempDir, "base.txt", "base commit")
		require.NoError(t, CheckoutBranch("feature"))

		needsMerge, err := NeedsMerge(base)
		require.NoError(t, err)
		assert.True(t, needsMerge, "the working branch does not contain the advanced base tip")
	})

	t.Run("diverged base needs a merge", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		base, err := GetCurrentBranch()
		require.NoError(t, err)

		require.NoError(t, CreateBranch("feature"))
		writeAndCommit(t, tempDir, "feature.txt", "feature commit")
		require.NoError(t, CheckoutBranch(base))
		writeAndCommit(t, tempDir, "base.txt", "base commit")
		require.NoError(t, CheckoutBranch("feature"))

		needsMerge, err := NeedsMerge(base)
		require.NoError(t, err)
		assert.True(t, needsMerge, "the working branch and the base branch have diverged")
	})

	t.Run("missing base needs no merge", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		t.Chdir(tempDir)

		needsMerge, err := NeedsMerge("does-not-exist")
		require.NoError(t, err)
		assert.False(t, needsMerge, "a base branch that cannot be resolved needs no merge")
	})
}

// writeAndCommit stages a file with the given content and commits it on the
// current branch.
func writeAndCommit(t *testing.T, dir, name, message string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(message), 0644))
	require.NoError(t, StageFile(name))
	require.NoError(t, Commit(message))
}

func TestAbortMerge(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)

	featureBranch := "abort-feature"
	err = CheckoutOrCreateBranch(featureBranch)
	require.NoError(t, err)

	conflictFile := "abort-conflict.txt"
	err = os.WriteFile(filepath.Join(tempDir, conflictFile), []byte("feature content"), 0644)
	require.NoError(t, err)
	err = StageFile(conflictFile)
	require.NoError(t, err)
	err = Commit("feature abort commit")
	require.NoError(t, err)

	_, err = runGit("checkout", base)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, conflictFile), []byte("base content"), 0644)
	require.NoError(t, err)
	err = StageFile(conflictFile)
	require.NoError(t, err)
	err = Commit("base abort commit")
	require.NoError(t, err)

	_ = Merge(featureBranch)

	err = AbortMerge()
	require.NoError(t, err)

	_, err = runGit("rev-parse", "--verify", "MERGE_HEAD")
	require.Error(t, err, "MERGE_HEAD should not exist after abort")
}

func TestAbortMerge_NoMergeInProgress(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	err := AbortMerge()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to abort merge")
}
