package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRunIterationStartsAndStopsServicesEachIteration(t *testing.T) {
	svcMock := &mockServices{}
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2)),
		withServices(svcMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.Equal(t, 2, svcMock.startCount)
	require.Equal(t, 2, svcMock.stopCount)
	require.Equal(t, 2, svcMock.removeLogsCount)
}

func TestRunIterationServiceStartupFailureTriggersFix(t *testing.T) {
	aiMock := &mockAI{}
	runner := withMocks(
		withServices(servicesThatFailToStart()),
		withProject(project.ThatReportsIncompleteUntil(1)),
		withAI(aiMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.True(t, aiServiceFixCalled(runner))
	require.Equal(t, 1, aiPickCalls(runner))
}

func TestRunIterationServiceFixFailureReturnsError(t *testing.T) {
	runner := withMocks(
		withServices(servicesThatFailToStart()),
		withAI(aiThatFailsServiceFix()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.Zero(t, aiPickCalls(runner))
}

func TestRunIterationPassesSelectedItemToDeveloper(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(1)),
		withAI(aiThatPicksIndex(2)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(4)), config.Any())
	require.NoError(t, err)
	require.Equal(t, 2, aiLastDevelopedIndex(runner))
}

func TestRunIterationLeavesProjectFileUntouched(t *testing.T) {
	projMock := project.ThatReportsIncompleteUntil(2)
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, projMock.Written())
}
