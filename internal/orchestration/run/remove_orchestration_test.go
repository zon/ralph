package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRemoveOrchestrationSkipsWhenNoSpec(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassingWithNoSpec()),
	)
	err := runner.RunLocal(project.WithAllPassing(), config.Any())
	require.NoError(t, err)
	require.False(t, gitOrchestrationRemovalCommitted(runner))
}

func TestRemoveOrchestrationSkipsWhenNoOrchestration(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassingWithSpecButNoOrchestration()),
	)
	err := runner.RunLocal(project.WithAllPassing(), config.Any())
	require.NoError(t, err)
	require.False(t, gitOrchestrationRemovalCommitted(runner))
}

func TestRemoveOrchestrationRemovesAndCommitsWhenPresent(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassingWithOrchestration()),
	)
	err := runner.RunLocal(project.WithAllPassing(), config.Any())
	require.NoError(t, err)
	require.True(t, projectOrchestrationRemoved(runner))
	require.True(t, gitOrchestrationRemovalCommitted(runner))
}

func TestRemoveOrchestrationFailureSendsErrorNotification(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassingWithOrchestrationRemovalFailure()),
	)
	err := runner.RunLocal(project.WithAllPassing(), config.Any())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
	require.False(t, githubPRCreated(runner))
}
