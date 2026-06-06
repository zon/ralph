package argo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListResolvesContextAndCallsArgo(t *testing.T) {
	cmd := argo.withMocks()
	err := cmd.List(flags.anyList())
	require.NoError(t, err)
	require.True(t, argoClient.listCalled())
}

func TestListPropagatesContextResolutionFailure(t *testing.T) {
	cmd := argo.withMocks(
		argo.withContext(ctx.thatFails()),
	)
	err := cmd.List(flags.anyList())
	require.Error(t, err)
	require.False(t, argoClient.listCalled())
}

func TestStopResolvesContextAndCallsArgo(t *testing.T) {
	cmd := argo.withMocks()
	err := cmd.Stop(flags.anyStop())
	require.NoError(t, err)
	require.True(t, argoClient.stopCalled())
}

func TestStopPropagatesContextResolutionFailure(t *testing.T) {
	cmd := argo.withMocks(
		argo.withContext(ctx.thatFails()),
	)
	err := cmd.Stop(flags.anyStop())
	require.Error(t, err)
	require.False(t, argoClient.stopCalled())
}
