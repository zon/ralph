package loop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loopWithSync builds an in-process loop command whose git and AI clients are
// the supplied mocks, so the base-branch synchronization scenarios can assert
// what the loop did without touching a real repository or AI.
func loopWithSync(git *mockGitClient, ai *mockAIClient, output OutputClient) *Cmd {
	return NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{},
		ai,
		&mockReportReader{reports: nothingToDoReports()},
		git,
		&mockPullRequestOpener{},
		envNotInWorkflow(),
		WithOutput(output),
	)
}

// TestRunLocalSyncsBaseBranchBeforeFirstIteration covers the "Merged before the
// first iteration" scenario: the loop fetches the branch the loop branch was
// created from and merges it into loop-<slug> before the first iteration runs.
func TestRunLocalSyncsBaseBranchBeforeFirstIteration(t *testing.T) {
	git := &mockGitClient{currentBranch: "main", needsMerge: true}
	ai := &mockAIClient{}
	cmd := loopWithSync(git, ai, &mockOutput{})

	result, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, git.fetchBranchCalled, "the base branch is fetched before the first iteration")
	assert.True(t, git.mergeCalled, "the base branch is merged before the first iteration")
	assert.Equal(t, "main", git.lastFetchedBranch, "the branch the loop branch was created from is fetched")
	assert.Equal(t, "main", git.lastMergedBranch, "local mode merges the local base branch")
	assert.NotZero(t, ai.calls, "the first iteration runs after synchronization")
}

// TestRunLocalSyncSkippedWhenBaseUnset asserts no fetch or merge happens when
// the branch the loop branch was created from is unknown.
func TestRunLocalSyncSkippedWhenBaseUnset(t *testing.T) {
	git := &mockGitClient{}
	ai := &mockAIClient{}
	cmd := loopWithSync(git, ai, &mockOutput{})

	_, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.NoError(t, err)
	assert.False(t, git.fetchBranchCalled, "no base branch is fetched when none is resolved")
	assert.False(t, git.mergeCalled, "no merge is attempted when no base branch is resolved")
}

// TestRunLocalSkipsMergeWhenBaseAlreadyContained covers the "Branch is
// up-to-date" scenario: the base branch is fetched but no merge runs when its
// tip is already contained in the loop branch.
func TestRunLocalSkipsMergeWhenBaseAlreadyContained(t *testing.T) {
	git := &mockGitClient{currentBranch: "main", needsMerge: false}
	ai := &mockAIClient{}
	cmd := loopWithSync(git, ai, &mockOutput{})

	_, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.NoError(t, err)
	assert.True(t, git.fetchBranchCalled, "the base branch is still fetched")
	assert.False(t, git.mergeCalled, "an up-to-date base branch is not merged")
}

// TestRunLocalFetchFailureWarnsAndContinues covers the "Base branch fetch
// failure" scenario: a fetch failure logs a warning and the loop runs without
// merging.
func TestRunLocalFetchFailureWarnsAndContinues(t *testing.T) {
	out := &mockOutput{}
	git := &mockGitClient{currentBranch: "main", fetchErr: errors.New("fetch boom")}
	ai := &mockAIClient{}
	cmd := loopWithSync(git, ai, out)

	_, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.NoError(t, err)
	assert.NotEmpty(t, out.warnings, "a fetch failure is warned about")
	assert.False(t, git.mergeCalled, "a failed fetch skips the merge")
	assert.NotZero(t, ai.calls, "the loop continues without merging")
}

// TestRunLocalConflictAbortsAndResolvesWithAI covers the "Conflicts resolved by
// AI" scenario: a conflicting merge is aborted and the configured agent is
// asked to resolve it against the loop branch.
func TestRunLocalConflictAbortsAndResolvesWithAI(t *testing.T) {
	git := &mockGitClient{currentBranch: "main", needsMerge: true, mergeErr: errors.New("conflict")}
	ai := &mockAIClient{}
	cmd := loopWithSync(git, ai, &mockOutput{})

	_, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.NoError(t, err)
	assert.True(t, git.abortMergeCalled, "the conflicting merge is aborted")
	assert.True(t, ai.resolveConflictsCalled, "the configured agent resolves the conflict")
	assert.Equal(t, "main", ai.lastResolveBase, "the agent resolves the same base branch that conflicted")
	assert.Equal(t, "loop-fmt", ai.lastResolveProject, "the agent resolves conflicts on the loop branch")
	assert.NotZero(t, ai.calls, "the loop continues after conflict resolution")
}

// TestRunLocalConflictResolutionFailureAbortsRun covers the "Conflict
// resolution failure aborts the run" scenario: a failed resolution returns an
// error and no iteration runs.
func TestRunLocalConflictResolutionFailureAbortsRun(t *testing.T) {
	resolveErr := errors.New("resolution boom")
	git := &mockGitClient{currentBranch: "main", needsMerge: true, mergeErr: errors.New("conflict")}
	ai := &mockAIClient{resolveConflictsErr: resolveErr}
	pr := &mockPullRequestOpener{}
	cmd := NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{},
		ai,
		&mockReportReader{reports: nothingToDoReports()},
		git,
		pr,
		envNotInWorkflow(),
		WithOutput(&mockOutput{}),
	)

	_, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.Error(t, err)
	assert.Equal(t, resolveErr, err, "the resolution error is returned unchanged")
	assert.Zero(t, ai.calls, "a failed resolution stops the loop before any iteration")
	assert.Zero(t, pr.calls, "a failed resolution opens no pull request")
}

// TestRunWorktreeSyncMergesRemoteBase covers the worktree merge ref: inside a
// worktree the fetched remote-tracking base is merged, because the base branch
// is normally checked out in the main checkout and cannot be moved.
func TestRunWorktreeSyncMergesRemoteBase(t *testing.T) {
	git := &mockGitClient{currentBranch: "main", needsMerge: true}
	ai := &mockAIClient{}
	cmd := loopWithSync(git, ai, &mockOutput{})
	cmd.base = "main"

	err := cmd.RunResolvedInWorktree(&Result{Slug: "fmt", Steps: []string{"run gofmt"}}, 10)

	require.NoError(t, err)
	assert.True(t, git.mergeCalled, "the fetched remote base is merged inside the worktree")
	assert.Equal(t, "origin/main", git.lastMergedBranch, "the worktree merges the fetched remote base")
	assert.Zero(t, git.switchCalls, "worktree execution leaves the current checkout on its branch")
}

// TestRunPropagatesCurrentBranchError asserts a failure to resolve the branch
// the loop branch was created from aborts the loop before it runs.
func TestRunPropagatesCurrentBranchError(t *testing.T) {
	branchErr := errors.New("detached HEAD")
	git := &mockGitClient{currentBranchErr: branchErr}
	ai := &mockAIClient{}
	cmd := loopWithSync(git, ai, &mockOutput{})

	result, err := cmd.Run("fmt", []string{"run gofmt"}, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the base branch cannot be resolved")
	assert.Equal(t, branchErr, err, "the current-branch error is returned unchanged")
	assert.Zero(t, ai.calls, "the loop does not run when the base branch cannot be resolved")
}
