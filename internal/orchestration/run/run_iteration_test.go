package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRunIterationStartsAndStopsServicesEachIteration(t *testing.T) {
	svcMock := &mockServicesClient{}
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(2)),
		withServices(svcMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.NoError(t, err)
	require.Equal(t, 2, svcMock.startCount)
	require.Equal(t, 2, svcMock.stopCount)
	require.Equal(t, 2, svcMock.removeLogsCount)
}

func TestRunIterationServiceStartupFailureTriggersFix(t *testing.T) {
	aiMock := &mockAIClient{}
	runner := withMocks(
		withServices(newServicesThatFailToStart()),
		withProject(newProjectThatReportsPassingAfterIterations(1)),
		withAI(aiMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.NoError(t, err)
	require.True(t, aiMock.fixServiceCalled)
	require.Len(t, aiMock.pickCalls, 1)
}

func TestRunIterationServiceFixFailureReturnsError(t *testing.T) {
	runner := withMocks(
		withServices(newServicesThatFailToStart()),
		withAI(newAIThatFailsServiceFix()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.Error(t, err)
	require.Empty(t, aiPickCalls(runner))
}
