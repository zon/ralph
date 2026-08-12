package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestPRScenario_AllItemsCompleteAfterIterationsCreatesPR(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2)),
		withGit(gitThatCommitsAhead()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.True(t, githubPRCreated(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}

func TestPRScenario_NoCommitsAheadSkipsPR(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, githubPRCreated(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}

func TestPRScenario_IterationLimitReachedSkipsPR(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(1)), config.WithExtraIterations(0))
	require.Error(t, err)
	require.False(t, githubCreatePRCalled(runner))
	require.NotEmpty(t, notifyErrors(runner))
}
