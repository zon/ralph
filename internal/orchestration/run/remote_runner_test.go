package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/workflow"
)

func TestRunRemoteBranchNotPushed(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteGit(git.ThatReportsBranchNotPushed()),
	)
	err := runner.RunRemote(project.Any(), false)
	require.Error(t, err)
	require.False(t, remoteWorkflowSubmitted(runner))
}

func TestRunRemoteBranchNotInSync(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteGit(git.ThatReportsBranchNotInSync()),
	)
	err := runner.RunRemote(project.Any(), false)
	require.Error(t, err)
	require.False(t, remoteWorkflowSubmitted(runner))
}

func TestRunRemoteWorkflowSubmissionFailure(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteWorkflow(workflow.ThatFailsOnSubmit()),
	)
	err := runner.RunRemote(project.Any(), false)
	require.Error(t, err)
}

func TestRunRemoteNoFollowPrintsLogHint(t *testing.T) {
	runner := withRemoteMocks()
	err := runner.RunRemote(project.Any(), false)
	require.NoError(t, err)
	require.True(t, remoteWorkflowLogHintPrinted(runner))
	require.Empty(t, remoteNotifySuccesses(runner))
}

func TestRunRemoteFollowSuccess(t *testing.T) {
	runner := withRemoteMocks()
	err := runner.RunRemote(project.Any(), true)
	require.NoError(t, err)
	require.NotEmpty(t, remoteNotifySuccesses(runner))
}

func TestRunRemoteFollowFailureSendsErrorNotification(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteWorkflow(workflow.ThatFailsOnFollow()),
	)
	err := runner.RunRemote(project.Any(), true)
	require.Error(t, err)
	require.NotEmpty(t, remoteNotifyErrors(runner))
}
