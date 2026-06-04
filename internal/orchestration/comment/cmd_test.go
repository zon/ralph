package comment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunMissingCommentBodyAbortsBeforeWorkspace(t *testing.T) {
	cmd := comment.withMocks()
	err := cmd.Run(flags.withNoCommentBody())
	require.Error(t, err)
	require.False(t, workspace.setupCalled())
}

func TestRunWorkspaceFailureAbortsEarly(t *testing.T) {
	cmd := comment.withMocks(
		comment.withWorkspace(workspace.thatFailsSetup()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, config.loadCalled())
}

func TestRunServicesStartedBeforeAgentAndStopped(t *testing.T) {
	cmd := comment.withMocks()
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.Equal(t, 1, svcs.startCount())
	require.Equal(t, 1, svcs.stopCount())
	require.True(t, svcs.startedBeforeAgent())
}

func TestRunNoServicesFlagSkipsServiceStartup(t *testing.T) {
	cmd := comment.withMocks()
	err := cmd.Run(flags.withNoServices())
	require.NoError(t, err)
	require.Equal(t, 0, svcs.startCount())
}

func TestRunChangesCommittedAndPushed(t *testing.T) {
	cmd := comment.withMocks(
		comment.withGit(git.withChangesAndReport()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, git.committedAndPushed())
}

func TestRunNoChangesSkipsCommit(t *testing.T) {
	cmd := comment.withMocks(
		comment.withGit(git.withNoChanges()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.False(t, git.committedAndPushed())
}

func TestRunReplyPostedAfterCommit(t *testing.T) {
	cmd := comment.withMocks(
		comment.withGit(git.withChangesAndReport()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, github.commentPosted())
}

func TestRunReplyPostedWhenNoChanges(t *testing.T) {
	cmd := comment.withMocks(
		comment.withGit(git.withNoChanges()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, github.commentPosted())
}
