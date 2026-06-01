package run

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/workflow"
)

func TestRunRemoteBranchNotPushed(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteGit(&git.MockClient{
			IsBranchSyncedWithRemoteFunc: func(branch string) error {
				return fmt.Errorf("branch '%s' has not been pushed to remote - please push before running remotely", branch)
			},
		}),
	)
	err := runner.RunRemote(project.Any(), false)
	require.Error(t, err)
	require.False(t, remoteWorkflowSubmitted(runner))
}

func TestRunRemoteBranchNotInSync(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteGit(&git.MockClient{
			IsBranchSyncedWithRemoteFunc: func(branch string) error {
				return fmt.Errorf("branch '%s' is not in sync with remote - please push your changes before running remotely", branch)
			},
		}),
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
