package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRemoveOrchestrationSkipsWhenNoSpec(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete().WithNoSpec()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, gitOrchestrationRemovalCommitted(runner))
}

func TestRemoveOrchestrationSkipsWhenNoOrchestration(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete().WithSpecButNoOrchestration()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, gitOrchestrationRemovalCommitted(runner))
}

func TestRemoveOrchestrationRemovesAndCommitsWhenPresent(t *testing.T) {
	projMock := project.ThatReportsAllComplete().WithOrchestration()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.True(t, projMock.OrchestrationRemoved())
	require.True(t, gitOrchestrationRemovalCommitted(runner))
}

func TestRemoveOrchestrationFailureSendsErrorNotification(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete().WithOrchestration().ThatFailsRemoval()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
	require.False(t, githubCreatePRCalled(runner))
}
