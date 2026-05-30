package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRunLocalBeforeCommandFailureAbortsEarly(t *testing.T) {
	runner := withMocks(
		withServices(newServicesThatFailBeforeCommands()),
	)
	err := runner.RunLocal(project.Any(), config.Any())
	require.Error(t, err)
	require.False(t, gitBranchSwitched(runner))
}

func TestRunLocalIterationFailureSendsErrorNotification(t *testing.T) {
	runner := withMocks(
		withAI(newAIThatAlwaysFails()),
	)
	err := runner.RunLocal(project.WithFailingRequirements(), config.Any())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
}

func TestRunLocalAllRequirementsPassCreatesPR(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassing()),
		withGitHub(newGitHubWithCommitsAhead()),
	)
	err := runner.RunLocal(project.WithAllPassing(), config.Any())
	require.NoError(t, err)
	require.True(t, githubPRCreated(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}

func TestRunLocalStatsPrintedOnSuccess(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.WithAllPassing(), config.Any())
	require.NoError(t, err)
	require.True(t, aiStatsPrinted(runner))
}

func TestRunLocalStatsPrintedOnFailure(t *testing.T) {
	runner := withMocks(
		withAI(newAIThatAlwaysFails()),
	)
	err := runner.RunLocal(project.WithFailingRequirements(), config.Any())
	require.Error(t, err)
	require.True(t, aiStatsPrinted(runner))
}

func TestRunLocalNoCommitsSkipsPR(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.WithAllPassing(), config.Any())
	require.NoError(t, err)
	require.False(t, githubPRCreated(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}
