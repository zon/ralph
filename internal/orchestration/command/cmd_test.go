package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowCommandRunMissingCommandAbortsBeforeWorkspace(t *testing.T) {
	cmd := workflowCommand.withMocks()
	err := cmd.Run(flags.withNoCommand())
	require.Error(t, err)
	require.False(t, workspace.setupCalled())
}

func TestWorkflowCommandRunWorkspaceFailureAbortsEarly(t *testing.T) {
	cmd := workflowCommand.withMocks(
		workflowCommand.withWorkspace(workspace.thatFailsSetup()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, exec.runCalled())
}

func TestWorkflowCommandRunExecutesCommandAfterWorkspace(t *testing.T) {
	cmd := workflowCommand.withMocks()
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, exec.runCalled())
}
