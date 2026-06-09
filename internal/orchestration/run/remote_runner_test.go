package run

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/workflow"
)

func TestRunBranchNotPushedAbortsBeforeSubmit(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteGit(&git.MockClient{
			IsBranchSyncedWithRemoteFunc: func(branch string) error {
				return fmt.Errorf("branch '%s' has not been pushed to remote - please push before running remotely", branch)
			},
		}),
	)
	err := runner.Run(project.ForProjectInput(project.Any()), runRemoteFlagsAny())
	require.Error(t, err)
	require.False(t, remoteWorkflowSubmitted(runner))
}

func TestRunBranchNotInSyncAbortsBeforeSubmit(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteGit(&git.MockClient{
			IsBranchSyncedWithRemoteFunc: func(branch string) error {
				return fmt.Errorf("branch '%s' is not in sync with remote - please push your changes before running remotely", branch)
			},
		}),
	)
	err := runner.Run(project.ForProjectInput(project.Any()), runRemoteFlagsAny())
	require.Error(t, err)
	require.False(t, remoteWorkflowSubmitted(runner))
}

func TestRunSubmitFailureReturnsError(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteWorkflow(workflow.ThatFailsOnSubmit()),
	)
	err := runner.Run(project.ForProjectInput(project.Any()), runRemoteFlagsAny())
	require.Error(t, err)
}

func TestRunNoFollowPrintsLogHint(t *testing.T) {
	runner := withRemoteMocks()
	err := runner.Run(project.ForProjectInput(project.Any()), runRemoteFlagsWithoutFollow())
	require.NoError(t, err)
	require.True(t, remoteWorkflowLogHintPrinted(runner))
	require.False(t, remoteWorkflowFollowLogsCalled(runner))
}

func TestRunFollowStreamsLogsAndNotifiesSuccess(t *testing.T) {
	runner := withRemoteMocks()
	err := runner.Run(project.ForProjectInput(project.Any()), runRemoteFlagsWithFollow())
	require.NoError(t, err)
	require.True(t, remoteWorkflowFollowLogsCalled(runner))
	require.True(t, remoteNotifySuccessSent(runner))
}

func TestRunFollowFailureNotifiesErrorAndReturns(t *testing.T) {
	runner := withRemoteMocks(
		withRemoteWorkflow(workflow.ThatFailsOnFollow()),
	)
	err := runner.Run(project.ForProjectInput(project.Any()), runRemoteFlagsWithFollow())
	require.Error(t, err)
	require.True(t, remoteNotifyErrorSent(runner))
}

func TestRunDebugBranchPassedToSubmit(t *testing.T) {
	runner := withRemoteMocks()
	err := runner.Run(project.ForProjectInput(project.Any()), runRemoteFlagsWithDebug("my-fix"))
	require.NoError(t, err)
	require.Equal(t, "my-fix", remoteWorkflowLastDebugBranch(runner))
}
