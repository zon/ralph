package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

// TestRunLocalNeverRemovesOrchestration asserts a run that finishes its items
// never deletes an orchestration document, even when the project still
// references one.
func TestRunLocalNeverRemovesOrchestration(t *testing.T) {
	projMock := project.ThatReportsAllComplete().WithOrchestration()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, projMock.OrchestrationRemoved(), "the run must not delete an orchestration document")
	require.False(t, gitOrchestrationRemovalCommitted(runner), "the run must not commit an orchestration removal")
}

// TestRunLocalIgnoresOrchestrationRemovalFailure asserts the run completes even
// though the project reports that removing its orchestration document would
// fail: the removal is never attempted, so it cannot abort the run.
func TestRunLocalIgnoresOrchestrationRemovalFailure(t *testing.T) {
	projMock := project.ThatReportsAllComplete().WithOrchestration().ThatFailsRemoval()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, projMock.OrchestrationRemoved())
	require.False(t, gitOrchestrationRemovalCommitted(runner))
}

// TestRunLocalRemovesProjectFileWithoutOrchestrationCommit asserts the run's own
// cleanup still deletes the project file, and that no orchestration removal is
// committed in its place.
func TestRunLocalRemovesProjectFileWithoutOrchestrationCommit(t *testing.T) {
	projMock := project.ThatReportsAllComplete().WithOrchestration()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithCleanup())
	require.NoError(t, err)
	require.True(t, projMock.Removed())
	require.False(t, gitOrchestrationRemovalCommitted(runner))
}
