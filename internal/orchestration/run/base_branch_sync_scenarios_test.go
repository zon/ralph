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

// TestRunInWorkflowSyncsDeliveredBaseBeforeFirstIteration covers the
// "Synchronization in a workflow container" scenario: when the run executes
// inside the workflow container, the start-of-run synchronization fetches and
// merges the base branch delivered to the container before the first iteration.
func TestRunInWorkflowSyncsDeliveredBaseBeforeFirstIteration(t *testing.T) {
	runner := withMocks(
		withEnv(envInWorkflow()),
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

// TestRunLocalSyncsBaseBranchAgainBeforePR covers the "Synchronized again before
// the pull request" scenario: once the work is complete, the base branch is
// fetched and merged again and the merge is pushed before the pull request is
// opened.
func TestRunLocalSyncsBaseBranchAgainBeforePR(t *testing.T) {
	g := gitThatNeedsMerge()
	gh := &mockGitHub{}
	gh.createPRFunc = func(*project.Project, string) error {
		require.True(t, g.pushCalled, "the merge is pushed before the pull request is opened")
		return nil
	}
	runner := withMocks(
		withGit(g),
		withGitHub(gh),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.Equal(t, 2, gitFetchCalls(runner), "the base branch is fetched before the first iteration and again before the pull request")
	require.Equal(t, 2, gitMergeCalls(runner), "the base branch is merged before the first iteration and again before the pull request")
	require.True(t, gitPushCalled(runner))
	require.True(t, githubCreatePRCalled(runner))
}

func TestRunLocalPRSyncSkipsPushWhenBaseAlreadyContained(t *testing.T) {
	runner := withMocks(
		withGit(gitNewMock()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.False(t, gitMergeCalled(runner))
	require.False(t, gitPushCalled(runner), "an up-to-date base branch leaves nothing to push")
	require.True(t, githubCreatePRCalled(runner))
}

func TestRunLocalPRSyncFetchFailureWarnsAndStillCreatesPR(t *testing.T) {
	runner := withMocks(
		withGit(gitThatFailsFetch()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.NotEmpty(t, outputWarnings(runner))
	require.False(t, gitPushCalled(runner))
	require.True(t, githubCreatePRCalled(runner), "a fetch failure does not stop the pull request")
}

func TestRunLocalPRSyncPushFailureAbortsBeforePR(t *testing.T) {
	runner := withMocks(
		withGit(gitThatFailsPush()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.Error(t, err)
	require.True(t, gitPushCalled(runner))
	require.False(t, githubCreatePRCalled(runner), "a failed push stops the run before the pull request is opened")
	require.NotEmpty(t, notifyErrors(runner))
}

func TestRunLocalPRConflictResolvedThenPushed(t *testing.T) {
	runner := withMocks(
		withGit(gitThatConflictsOnSecondMerge()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.True(t, gitMergeAborted(runner), "the pre-pull-request merge conflict is aborted")
	require.True(t, aiResolveConflictsCalled(runner))
	require.True(t, gitPushCalled(runner), "the resolved merge is pushed before the pull request")
	require.True(t, githubCreatePRCalled(runner))
}

func TestRunLocalPRConflictResolutionFailureSkipsPR(t *testing.T) {
	runner := withMocks(
		withGit(gitThatConflictsOnSecondMerge()),
		withAI(&mockAI{
			resolveConflictsFunc: func(string, string) error { return errNonFatal },
		}),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.Error(t, err)
	require.True(t, gitMergeAborted(runner))
	require.False(t, gitPushCalled(runner))
	require.False(t, githubCreatePRCalled(runner), "a failed resolution opens no pull request")
	require.NotEmpty(t, notifyErrors(runner))
}

func TestRunLocalInWorktreeSyncsBaseBranchAgainBeforePR(t *testing.T) {
	runner := withMocks(
		withGit(gitThatNeedsMerge()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocalInWorktree(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.Equal(t, 2, gitMergeCalls(runner))
	require.Equal(t, "origin/main", gitLastMergedBranch(runner), "the worktree merges the fetched remote base again before the pull request")
	require.True(t, gitPushCalled(runner))
	require.True(t, githubCreatePRCalled(runner))
	require.False(t, gitBranchSwitched(runner))
}

func TestRunInWorkflowSyncsDeliveredBaseAgainBeforePR(t *testing.T) {
	runner := withMocks(
		withEnv(envInWorkflow()),
		withGit(gitThatNeedsMerge()),
		withProject(project.ThatReportsIncompleteUntil(1)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.Equal(t, 2, gitFetchCalls(runner), "the container syncs the delivered base again before the pull request")
	require.Equal(t, 2, gitMergeCalls(runner))
	require.True(t, gitPushCalled(runner))
	require.True(t, githubCreatePRCalled(runner))
}
