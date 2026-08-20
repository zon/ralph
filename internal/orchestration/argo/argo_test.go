package argo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListResolvesContextAndCallsArgo(t *testing.T) {
	cmd := argo.withMocks()
	err := cmd.List(flags.anyList())
	require.NoError(t, err)
	require.Equal(t, ctx.any(), argoClient.listContext())
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
	require.Equal(t, ctx.any(), argoClient.stopContext())
}

func TestStopPropagatesContextResolutionFailure(t *testing.T) {
	cmd := argo.withMocks(
		argo.withContext(ctx.thatFails()),
	)
	err := cmd.Stop(flags.anyStop())
	require.Error(t, err)
	require.False(t, argoClient.stopCalled())
}

func TestLogsWithWorkflowNameCallsArgoLogs(t *testing.T) {
	cmd := argo.withMocks()
	err := cmd.Logs(flags.anyLogs())
	require.NoError(t, err)
	require.Equal(t, ctx.any(), argoClient.logsContext())
	require.Equal(t, flags.anyLogs().WorkflowName, argoClient.loggedWorkflow())
	require.False(t, argoClient.loggedFollow())
}

func TestLogsFollowPassesFollowThrough(t *testing.T) {
	cmd := argo.withMocks()
	err := cmd.Logs(flags.followingLogs())
	require.NoError(t, err)
	require.True(t, argoClient.logsCalled())
	require.True(t, argoClient.loggedFollow())
}

func TestLogsWithoutNameResolvesTopWorkflow(t *testing.T) {
	cmd := argo.withMocks(
		argo.withWorkflowNames("ralph-b", "ralph-a"),
	)
	err := cmd.Logs(flags.logsWithoutName())
	require.NoError(t, err)
	require.True(t, argoClient.logsCalled())
	require.Equal(t, "ralph-b", argoClient.loggedWorkflow())
	require.True(t, argoClient.listWorkflowNamesCalled())
	require.Equal(t, ctx.any(), argoClient.listWorkflowNamesContext())
}

func TestLogsWithoutNameAndEmptyListReturnsErrNoWorkflows(t *testing.T) {
	cmd := argo.withMocks(
		argo.withNoWorkflowNames(),
	)
	err := cmd.Logs(flags.logsWithoutName())
	require.ErrorIs(t, err, ErrNoWorkflows)
	require.False(t, argoClient.logsCalled())
}

func TestLogsPropagatesContextResolutionFailure(t *testing.T) {
	cmd := argo.withMocks(
		argo.withContext(ctx.thatFails()),
	)
	err := cmd.Logs(flags.anyLogs())
	require.Error(t, err)
	require.False(t, argoClient.logsCalled())
}

func TestLogsPropagatesListFailure(t *testing.T) {
	cmd := argo.withMocks(
		argo.withListNamesFailure(),
	)
	err := cmd.Logs(flags.logsWithoutName())
	require.Error(t, err)
	require.False(t, argoClient.logsCalled())
}

func TestLogsPropagatesArgoLogsFailure(t *testing.T) {
	cmd := argo.withMocks(
		argo.withLogsFailure(),
	)
	err := cmd.Logs(flags.anyLogs())
	require.Error(t, err)
	require.True(t, argoClient.logsCalled())
}
