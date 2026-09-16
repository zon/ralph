package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRunLocalStatsPrintedOnSuccess(t *testing.T) {
	runner := withMocks(
		withEnv(envInWorkflow()),
		withProject(project.ThatReportsAllComplete()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.True(t, aiStatsPrinted(runner))
}

func TestRunLocalStatsPrintedOnFailure(t *testing.T) {
	runner := withMocks(
		withEnv(envInWorkflow()),
		withAI(aiThatAlwaysFails()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.True(t, aiStatsPrinted(runner))
}

func TestRunLocalStatsNotPrintedWhenNotInWorkflow(t *testing.T) {
	runner := withMocks(
		withEnv(envNotInWorkflow()),
		withProject(project.ThatReportsAllComplete()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, aiStatsPrinted(runner))
}

func TestRunLocalBeforeCommandFailureAbortsEarly(t *testing.T) {
	runner := withMocks(
		withServices(servicesThatFailBeforeCommands()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.Any()), config.Any())
	require.Error(t, err)
	require.False(t, gitBranchSwitched(runner))
}

func TestRunLocalResolvesProjectFileDirectly(t *testing.T) {
	proj := project.Any()
	proj.Path = "projects/demo.yaml"
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(proj), config.Any())
	require.NoError(t, err)
	require.Equal(t, "projects/demo.yaml", projMock.LastPath(), "the run resolves the supplied project file directly")
	require.Equal(t, 1, projMock.ResolveCount())
}

func TestRunLocalResolvesItemsWithConfiguredQuery(t *testing.T) {
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithItems(".requirements"))
	require.NoError(t, err)
	require.Equal(t, ".requirements", projMock.LastQuery())
}

func TestRunLocalItemQueryYieldingNoItemsAborts(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatFailsResolution()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
	require.Zero(t, aiPickCalls(runner))
}

func TestRunLocalResolvesItemsOncePerRun(t *testing.T) {
	projMock := project.ThatReportsIncompleteUntil(3)
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(5)), config.Any())
	require.NoError(t, err)
	require.Equal(t, 1, projMock.ResolveCount())
}

func TestRunLocalIterationFailureSendsErrorNotification(t *testing.T) {
	runner := withMocks(
		withAI(aiThatAlwaysFails()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
}

func TestRunLocalAllItemsCompleteCreatesPR(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete()),
		withGit(gitThatCommitsAhead()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.True(t, githubPRCreated(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}

func TestRunLocalNoCommitsSkipsPR(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, githubPRCreated(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}
