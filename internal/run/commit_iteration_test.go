package run

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommitIterationUsesReportWhenPresent(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsPassingAfterIterations(1)),
		withGit(withChangesAndReport()),
	)
	err := runner.RunLocal(withFailingRequirements(), anyConfig())
	require.NoError(t, err)
	_, changelogCalls := agentCalls(runner)
	require.Zero(t, changelogCalls)
	require.True(t, hasCommitted(runner))
}

func TestCommitIterationGeneratesChangelogWhenNoReport(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsPassingAfterIterations(1)),
		withGit(withChangesButNoReport()),
	)
	err := runner.RunLocal(withFailingRequirements(), anyConfig())
	require.NoError(t, err)
	_, changelogCalls := agentCalls(runner)
	require.Equal(t, 1, changelogCalls)
	require.True(t, hasCommitted(runner))
}

func TestCommitIterationSkipsCommitWhenNoChanges(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsPassingAfterIterations(1)),
		withGit(withNoChanges()),
	)
	err := runner.RunLocal(withFailingRequirements(), anyConfig())
	require.NoError(t, err)
	_, changelogCalls := agentCalls(runner)
	require.Zero(t, changelogCalls)
	require.False(t, hasCommitted(runner))
}
