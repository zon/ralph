package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRemoveProjectFileSkippedWhenCleanupDisabled(t *testing.T) {
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, projMock.Removed())
	require.False(t, gitProjectRemovalCommitted(runner))
}

func TestRemoveProjectFileRemovesAndCommitsWhenCleanupEnabled(t *testing.T) {
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithCleanup())
	require.NoError(t, err)
	require.True(t, projMock.Removed())
	require.True(t, gitProjectRemovalCommitted(runner))
}

func TestRemoveProjectFileCommitsBeforePR(t *testing.T) {
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
		withGit(gitThatCommitsAhead()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithCleanup())
	require.NoError(t, err)
	require.True(t, gitProjectRemovalCommitted(runner))
	require.True(t, githubPRCreated(runner))
}

func TestRemoveProjectFileSkippedWhenIterationLimitReached(t *testing.T) {
	projMock := project.ThatAlwaysReportsIncomplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(1)), config.WithCleanup().WithExtraIterations(0))
	require.Error(t, err)
	require.False(t, projMock.Removed())
}

func TestRemoveProjectFileFailureSendsErrorNotification(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete().ThatFailsProjectRemoval()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithCleanup())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
	require.False(t, githubCreatePRCalled(runner))
}
