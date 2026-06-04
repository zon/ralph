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

func TestCommandRunMissingCommandReturnsError(t *testing.T) {
	cmd := command.withMocks()
	err := cmd.Run(cmdFlags.withNoCommand())
	require.Error(t, err)
	require.False(t, workflow.submitCalled())
}

func TestCommandRunSubmitsWorkflow(t *testing.T) {
	cmd := command.withMocks()
	err := cmd.Run(cmdFlags.any())
	require.NoError(t, err)
	require.True(t, workflow.submitCalled())
}

func TestCommandRunStreamsLogsByDefault(t *testing.T) {
	cmd := command.withMocks()
	err := cmd.Run(cmdFlags.any())
	require.NoError(t, err)
	require.True(t, workflow.streamLogsCalled())
}

func TestCommandRunNoFollowSkipsLogStreaming(t *testing.T) {
	cmd := command.withMocks()
	err := cmd.Run(cmdFlags.withNoFollow())
	require.NoError(t, err)
	require.True(t, workflow.submitCalled())
	require.False(t, workflow.streamLogsCalled())
}
