package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/project"
)

func TestRunLocalStatsPrintedOnSuccess(t *testing.T) {
	runner := withMocks(
		withEnv(newEnvInWorkflow()),
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.ForProjectInput(passingProject()), anyConfig())
	require.NoError(t, err)
	require.True(t, aiStatsPrinted(runner))
}

func TestRunLocalStatsPrintedOnFailure(t *testing.T) {
	runner := withMocks(
		withEnv(newEnvInWorkflow()),
		withAI(newAIThatAlwaysFails()),
	)
	err := runner.RunLocal(project.ForProjectInput(failingProject()), anyConfig())
	require.Error(t, err)
	require.True(t, aiStatsPrinted(runner))
}

func TestRunLocalStatsNotPrintedWhenNotInWorkflow(t *testing.T) {
	runner := withMocks(
		withEnv(newEnvNotInWorkflow()),
		withProject(newProjectThatReportsAllPassing()),
	)
	err := runner.RunLocal(project.ForProjectInput(passingProject()), anyConfig())
	require.NoError(t, err)
	require.False(t, aiStatsPrinted(runner))
}
