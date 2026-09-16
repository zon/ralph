package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRemoveProjectFileSkippedWhenCleanupDisabled(t *testing.T) {
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithoutCleanup())
	require.NoError(t, err)
	require.False(t, projMock.Removed())
	require.False(t, gitProjectRemovalCommitted(runner))
}

func TestRemoveProjectFileRemovesByDefault(t *testing.T) {
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.True(t, projMock.Removed(), "cleanup should be enabled by default")
	require.True(t, gitProjectRemovalCommitted(runner))
}

func TestRemoveProjectFileRemovesAndCommitsWhenCleanupEnabled(t *testing.T) {
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithCleanup())
	require.NoError(t, err)
	require.True(t, projMock.Removed())
	require.True(t, gitProjectRemovalCommitted(runner))
}

func TestRemoveProjectFileCommitsBeforePR(t *testing.T) {
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
		withGit(gitThatCommitsAhead()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithCleanup())
	require.NoError(t, err)
	require.True(t, gitProjectRemovalCommitted(runner))
	require.True(t, githubPRCreated(runner))
}

// TestRemoveProjectFileRunsAfterBaseSyncAsLastCommitBeforePR asserts the run's
// own cleanup lands after the pre-pull-request base-branch synchronization and
// immediately before the pull request is opened, so the deletion is the branch's
// last commit.
func TestRemoveProjectFileRunsAfterBaseSyncAsLastCommitBeforePR(t *testing.T) {
	g := gitThatNeedsMerge()
	gh := &mockGitHub{}
	gh.createPRFunc = func(*project.Project, string) error {
		require.True(t, g.commitProjectRemovalCalled, "the project deletion is committed before the pull request is opened")
		return nil
	}
	runner := withMocks(
		withGit(g),
		withGitHub(gh),
		withProject(project.ThatReportsAllComplete()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main").WithCleanup())
	require.NoError(t, err)

	order := gitEventOrder(runner)
	mergeIdx := gitEventIndex(order, "merge")
	removalIdx := gitEventIndex(order, "commit-project-removal")
	require.NotEqual(t, -1, mergeIdx)
	require.NotEqual(t, -1, removalIdx)
	require.Less(t, mergeIdx, removalIdx, "the base branch is synchronized before the project file is deleted")
	require.Less(t, gitEventIndex(order, "push"), removalIdx, "the base-branch merge is pushed before the project file is deleted")
	require.Equal(t, "commit-project-removal", order[len(order)-1], "the project file deletion is the last event before the pull request is opened")
	require.True(t, githubCreatePRCalled(runner))
}

func TestRemoveProjectFileSkippedWhenIterationLimitReached(t *testing.T) {
	projMock := project.ThatAlwaysReportsIncomplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(1)), config.WithCleanup().WithExtraIterations(0))
	require.Error(t, err)
	require.False(t, projMock.Removed())
}

func TestRemoveProjectFileFailureSendsErrorNotification(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete().ThatFailsProjectRemoval()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithCleanup())
	require.Error(t, err)
	require.NotEmpty(t, notifyErrors(runner))
	require.False(t, githubCreatePRCalled(runner))
}
