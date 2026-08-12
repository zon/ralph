package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitMessagesReturnsOnlyBranchCommits(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)

	require.NoError(t, CheckoutOrCreateBranch("feature-branch"))

	for _, file := range []string{"a.txt", "b.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, file), []byte(file), 0644))
		require.NoError(t, StageFile(file))
		require.NoError(t, Commit("feat: "+file))
	}

	messages, err := CommitMessages(base)
	require.NoError(t, err)

	require.Len(t, messages, 2)
	assert.Equal(t, "feat: b.txt\n", messages[0])
	assert.Equal(t, "feat: a.txt\n", messages[1])
}

func TestCommitMessagesExcludesBaseCommits(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "extra.txt"), []byte("extra"), 0644))
	require.NoError(t, StageFile("extra.txt"))
	require.NoError(t, Commit("feat: extra on base"))

	messages, err := CommitMessages(base)
	require.NoError(t, err)
	assert.Empty(t, messages, "commits on the base branch must not be reported")
}

func TestCommitMessagesEmptyWhenNoCommitsAhead(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)

	require.NoError(t, CheckoutOrCreateBranch("empty-branch"))

	messages, err := CommitMessages(base)
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestCommitMessagesPreservesFullMessageVerbatim(t *testing.T) {
	tempDir := setupTestRepo(t)
	t.Chdir(tempDir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)

	require.NoError(t, CheckoutOrCreateBranch("trailer-branch"))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "serializer.go"), []byte("package main\n"), 0644))
	require.NoError(t, StageFile("serializer.go"))
	require.NoError(t, Commit("feat: add serializer\n\nRalph item 0 (csv-serializer) completed"))

	messages, err := CommitMessages(base)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "feat: add serializer\n\nRalph item 0 (csv-serializer) completed\n", messages[0])
}
