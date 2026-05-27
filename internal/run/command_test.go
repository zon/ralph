package run

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunLocalBeforeCommandFailureAbortsEarly(t *testing.T) {
	runner := withMocks(
		withServices(thatFailBeforeCommands()),
	)
	err := runner.RunLocal(anyProject(), anyConfig())
	require.Error(t, err)
	require.False(t, hasSwitchedBranch(runner))
}

func TestRunLocalIterationFailureSendsErrorNotification(t *testing.T) {
	runner := withMocks(
		withAI(thatAlwaysFails()),
	)
	err := runner.RunLocal(withFailingRequirements(), anyConfig())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
}

func TestRunLocalAllRequirementsPassCreatesPR(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsAllPassing()),
		withGit(withCommitsAhead()),
	)
	err := runner.RunLocal(withAllPassing(), anyConfig())
	require.NoError(t, err)
	require.True(t, hasCreatedPR(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}

func TestRunLocalNoCommitsSkipsPR(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsAllPassing()),
	)
	err := runner.RunLocal(withAllPassing(), anyConfig())
	require.NoError(t, err)
	require.False(t, hasCreatedPR(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}

func TestRunLocalScenario_BranchSwitchedBeforeIteration(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsAllPassing()),
	)
	err := runner.RunLocal(withAllPassing(), anyConfig())
	require.NoError(t, err)
	require.True(t, hasSwitchedBranch(runner))
	require.Empty(t, iterateCalls(runner))
}

func TestRunLocalScenario_RequirementsPassAfterIterationsCreatesPR(t *testing.T) {
	runner := withMocks(
		withProject(thatReportsPassingAfterIterations(2)),
		withGit(withCommitsAhead()),
	)
	err := runner.RunLocal(withFailingRequirements(), anyConfig())
	require.NoError(t, err)
	require.True(t, hasCreatedPR(runner))
	require.NotEmpty(t, notifySuccesses(runner))
}
