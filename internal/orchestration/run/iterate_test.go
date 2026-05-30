package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestIterateExitsImmediatelyWhenAllPassing(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.WithAllPassing(), config.Any())
	require.NoError(t, err)
	require.Empty(t, aiIterateCalls(runner))
}

func TestIterateExitsEarlyWhenRequirementsPass(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(2)),
	)
	err := runner.RunLocal(project.WithFailingRequirements(), config.Any())
	require.NoError(t, err)
	require.Len(t, aiIterateCalls(runner), 2)
}

func TestIterateReturnsErrorAtMaxIterations(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatAlwaysReportsFailures()),
	)
	err := runner.RunLocal(project.WithMaxIterations(3), config.Any())
	require.Error(t, err)
	require.Len(t, aiIterateCalls(runner), 3)
}

func TestIterateStopsOnBlockedFile(t *testing.T) {
	runner := withMocks(
		withGit(newGitWithBlockedFile()),
	)
	err := runner.RunLocal(project.WithFailingRequirements(), config.Any())
	require.ErrorIs(t, err, ErrBlocked)
	require.Empty(t, aiIterateCalls(runner))
}

func TestIterateFatalAIErrorIsNotRetried(t *testing.T) {
	runner := withMocks(
		withAI(newAIThatReturnsFatalError()),
	)
	err := runner.RunLocal(project.WithFailingRequirements(), config.Any())
	require.Error(t, err)
	require.Len(t, aiIterateCalls(runner), 1)
	require.False(t, gitBlockedFileWritten(runner))
}

func TestIterateNonFatalAIErrorWritesBlockedFile(t *testing.T) {
	runner := withMocks(
		withAI(newAIThatReturnsNonFatalError()),
	)
	err := runner.RunLocal(project.WithFailingRequirements(), config.Any())
	require.Error(t, err)
	require.True(t, gitBlockedFileWritten(runner))
}
