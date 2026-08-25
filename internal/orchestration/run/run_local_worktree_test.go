package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRunLocalInWorktreeSkipsBranchSwitch(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete()),
	)
	err := runner.RunLocalInWorktree(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, gitBranchSwitched(runner), "worktree mode must not switch branches in the current checkout")
}

func TestRunLocalInWorktreeRunsFullLoop(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete()),
	)
	err := runner.RunLocalInWorktree(project.ForOrchestrationInput("specs/features/ralph/run/orchestration.md"), config.Any())
	require.NoError(t, err)
	require.False(t, gitBranchSwitched(runner))
	require.True(t, aiWriteProjectCalled(runner), "artifact generation runs in the worktree")
	require.True(t, gitArtifactsCommitted(runner))
}

func TestRunLocalInWorktreeIteratesAndCreatesPR(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(1)),
		withGit(gitThatCommitsAhead()),
	)
	err := runner.RunLocalInWorktree(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.Greater(t, aiPickCalls(runner), 0, "the loop iterates over items in the worktree")
	require.True(t, githubPRCreated(runner))
	require.False(t, gitBranchSwitched(runner))
}

func TestRunLocalInWorktreeFailureSkipsBranchSwitch(t *testing.T) {
	runner := withMocks(
		withAI(aiThatAlwaysFails()),
	)
	err := runner.RunLocalInWorktree(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.False(t, gitBranchSwitched(runner))
	require.NotEmpty(t, notifyErrors(runner))
}
