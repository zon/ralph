package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitAdapterFetchBranch_NoRemoteReturnsError(t *testing.T) {
	t.Parallel()

	adapter := &gitAdapter{}
	err := adapter.FetchBranch("nonexistent-branch")
	require.Error(t, err)
}

func TestGitAdapterNeedsMerge_NoBranchReturnsFalse(t *testing.T) {
	t.Parallel()

	adapter := &gitAdapter{}
	needs, err := adapter.NeedsMerge("nonexistent-branch")
	require.NoError(t, err)
	assert.False(t, needs)
}

func TestGitAdapterAbortMerge_NoMergeInProgressDoesNotPanic(t *testing.T) {
	t.Parallel()

	adapter := &gitAdapter{}
	adapter.AbortMerge()
}
