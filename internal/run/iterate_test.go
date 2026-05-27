package run

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIterateExitsImmediatelyWhenAllPassing(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsAllPassing()),
	)
	err := runner.RunLocal(withAllPassing(), anyConfig())
	require.NoError(t, err)
	require.Empty(t, iterateCalls(runner))
}

func TestIterateExitsEarlyWhenRequirementsPass(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsPassingAfterIterations(2)),
	)
	err := runner.RunLocal(withMaxIterations(10), anyConfig())
	require.NoError(t, err)
	require.Len(t, iterateCalls(runner), 2)
}

func TestIterateReturnsErrorAtMaxIterations(t *testing.T) {
	runner := withMocks(
		withProject(thatAlwaysReportsFailures()),
	)
	err := runner.RunLocal(withMaxIterations(3), anyConfig())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMaxIterationsReached)
	require.Contains(t, err.Error(), "1 requirements still failing")
	require.Len(t, iterateCalls(runner), 3)
}

func TestIterateStopsOnBlockedFile(t *testing.T) {
	runner := withMocks(
		withGit(withBlockedFile()),
	)
	err := runner.RunLocal(withFailingRequirements(), anyConfig())
	require.ErrorIs(t, err, ErrBlocked)
	require.Empty(t, iterateCalls(runner))
}

func TestIterateFatalAIErrorIsNotRetried(t *testing.T) {
	runner := withMocks(
		withAI(thatReturnsFatalError()),
	)
	err := runner.RunLocal(withFailingRequirements(), anyConfig())
	require.Error(t, err)
	require.Len(t, iterateCalls(runner), 1)
	require.False(t, hasWrittenBlocked(runner))
}

func TestIterateNonFatalAIErrorWritesBlockedFile(t *testing.T) {
	runner := withMocks(
		withAI(thatReturnsNonFatalError()),
	)
	err := runner.RunLocal(withFailingRequirements(), anyConfig())
	require.Error(t, err)
	require.True(t, hasWrittenBlocked(runner))
}
