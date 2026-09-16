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

func TestRunLocalInWorktreeResolvesProjectFileDirectly(t *testing.T) {
	proj := project.Any()
	proj.Path = "projects/demo.yaml"
	projMock := project.ThatReportsAllComplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocalInWorktree(project.ForProjectInput(proj), config.Any())
	require.NoError(t, err)
	require.Equal(t, "projects/demo.yaml", projMock.LastPath(), "worktree execution resolves the supplied project file directly")
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
