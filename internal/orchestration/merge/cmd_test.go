package merge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeWorkspaceFailureAbortsEarly(t *testing.T) {
	cmd := merge.withMocks(
		merge.withWorkspace(workspace.thatFailsSetup()),
	)
	err := cmd.Merge(flags.any())
	require.Error(t, err)
	require.False(t, project.loadAllCalled())
}

func TestMergeNoCompletedProjectsSkipsCleanupAndSync(t *testing.T) {
	cmd := merge.withMocks(
		merge.withProject(project.withNoCompletedProjects()),
	)
	err := cmd.Merge(flags.any())
	require.NoError(t, err)
	require.False(t, git.commitAndPushCalled())
	require.False(t, github.waitForHeadSyncCalled())
	require.True(t, github.mergePRCalled())
}

func TestMergeCompletedProjectsDeletedCommittedAndPushed(t *testing.T) {
	cmd := merge.withMocks(
		merge.withProject(project.withCompletedProjects()),
	)
	err := cmd.Merge(flags.any())
	require.NoError(t, err)
	require.True(t, project.deletedAll())
	require.True(t, git.commitAndPushCalled())
}

func TestMergeHeadSyncCalledAfterPush(t *testing.T) {
	cmd := merge.withMocks(
		merge.withProject(project.withCompletedProjects()),
	)
	err := cmd.Merge(flags.any())
	require.NoError(t, err)
	require.True(t, github.waitForHeadSyncCalled())
}

func TestMergeHeadSyncTimeoutAbortsBeforeMerge(t *testing.T) {
	cmd := merge.withMocks(
		merge.withProject(project.withCompletedProjects()),
		merge.withGitHub(github.thatTimesOutHeadSync()),
	)
	err := cmd.Merge(flags.any())
	require.Error(t, err)
	require.False(t, github.mergePRCalled())
}

func TestMergePRCalledOnSuccess(t *testing.T) {
	cmd := merge.withMocks()
	err := cmd.Merge(flags.any())
	require.NoError(t, err)
	require.True(t, github.mergePRCalled())
}
