package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRunLocalSyncsBaseBranchBeforeFirstIteration(t *testing.T) {
	runner := withMocks(
		withGit(gitThatNeedsMerge()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.True(t, gitFetchCalled(runner))
	require.True(t, gitMergeCalled(runner))
	require.Equal(t, "main", gitLastFetchedBranch(runner))
	require.Equal(t, "main", gitLastMergedBranch(runner))
	require.NotZero(t, aiPickCalls(runner), "the first iteration runs after synchronization")
}

func TestRunLocalSyncSkippedWhenBaseUnset(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, gitFetchCalled(runner))
	require.False(t, gitMergeCalled(runner))
}

func TestRunLocalSkipsMergeWhenBaseAlreadyContained(t *testing.T) {
	runner := withMocks(
		withGit(gitNewMock()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.True(t, gitFetchCalled(runner))
	require.False(t, gitMergeCalled(runner))
}

func TestRunLocalFetchFailureWarnsAndContinues(t *testing.T) {
	runner := withMocks(
		withGit(gitThatFailsFetch()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.NotEmpty(t, outputWarnings(runner))
	require.False(t, gitMergeCalled(runner))
	require.NotZero(t, aiPickCalls(runner), "the run continues without merging")
}

func TestRunLocalConflictAbortsMergeAndResolvesWithAI(t *testing.T) {
	runner := withMocks(
		withGit(gitThatConflicts()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.True(t, gitMergeAborted(runner))
	require.True(t, aiResolveConflictsCalled(runner))
	require.NotZero(t, aiPickCalls(runner), "the run continues after conflict resolution")
}

func TestRunLocalConflictResolutionFailureAbortsRun(t *testing.T) {
	runner := withMocks(
		withGit(gitThatConflicts()),
		withAI(&mockAI{
			resolveConflictsFunc: func(string, string) error { return errNonFatal },
		}),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.Error(t, err)
	require.True(t, gitMergeAborted(runner))
	require.Zero(t, aiPickCalls(runner), "a failed resolution stops the run before any iteration")
	require.NotEmpty(t, notifyErrors(runner))
}

func TestRunLocalInWorktreeSyncsBaseBranchBeforeFirstIteration(t *testing.T) {
	runner := withMocks(
		withGit(gitThatNeedsMerge()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocalInWorktree(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.True(t, gitFetchCalled(runner))
	require.True(t, gitMergeCalled(runner))
	require.Equal(t, "main", gitLastFetchedBranch(runner))
	require.Equal(t, "origin/main", gitLastMergedBranch(runner), "the fetched remote base is merged so a base branch checked out in the main checkout is not moved")
	require.False(t, gitBranchSwitched(runner), "worktree mode leaves the current checkout on its branch")
	require.NotZero(t, aiPickCalls(runner), "the first iteration runs after synchronization")
}

func TestRunLocalInWorktreeConflictResolvedWithFetchedBase(t *testing.T) {
	runner := withMocks(
		withGit(gitThatConflicts()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocalInWorktree(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.True(t, gitMergeAborted(runner))
	require.True(t, aiResolveConflictsCalled(runner))
	require.Equal(t, "origin/main", aiResolveBase(runner), "the agent merges the same fetched ref that conflicted")
	require.Equal(t, "test-project", aiResolveProject(runner))
	require.NotZero(t, aiPickCalls(runner), "the run continues after conflict resolution")
}
