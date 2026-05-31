package run

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunLocalStatsPrintedOnSuccess(t *testing.T) {
	runner := withMocks(
		withEnv(newEnvInWorkflow()),
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(passingProject(), anyConfig())
	require.NoError(t, err)
	require.True(t, aiStatsPrinted(runner))
}

func TestRunLocalStatsPrintedOnFailure(t *testing.T) {
	runner := withMocks(
		withEnv(newEnvInWorkflow()),
		withAI(newAIThatAlwaysFails()),
	)
	err := runner.RunLocal(failingProject(), anyConfig())
	require.Error(t, err)
	require.True(t, aiStatsPrinted(runner))
}

func TestRunLocalStatsNotPrintedWhenNotInWorkflow(t *testing.T) {
	runner := withMocks(
		withEnv(newEnvNotInWorkflow()),
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(passingProject(), anyConfig())
	require.NoError(t, err)
	require.False(t, aiStatsPrinted(runner))
}
