package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestRunLocalRestoresStartingBranchAfterPR(t *testing.T) {
	tests := []struct {
		name     string
		starting string
	}{
		{name: "run started on main returns to main", starting: "main"},
		{name: "run started on a feature branch returns to it", starting: "feature-x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := withMocks(
				withProject(project.ThatReportsAllComplete()),
				withGit(gitThatCommitsAhead()),
			)
			err := runner.RunLocal(
				project.ForProjectInput(project.Any()),
				config.Any().WithStartingBranch(tt.starting),
			)
			require.NoError(t, err)
			require.True(t, githubPRCreated(runner), "the run opens the pull request before restoring the checkout")
			require.True(t, gitCheckoutBranchCalled(runner), "the run checks out a branch after opening the pull request")
			require.Equal(t, tt.starting, gitLastCheckedOutBranch(runner))
		})
	}
}

func TestRunLocalStaysOnProjectBranchWhenItStartedThere(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete()),
		withGit(gitThatCommitsAhead()),
	)
	err := runner.RunLocal(
		project.ForProjectInput(project.Any()),
		config.Any().WithStartingBranch("test-project"),
	)
	require.NoError(t, err)
	require.False(t, gitCheckoutBranchCalled(runner), "no checkout is needed when the run started on the project branch")
}

func TestRunLocalStaysOnProjectBranchWhenNoPullRequestIsOpened(t *testing.T) {
	t.Run("items stay incomplete", func(t *testing.T) {
		runner := withMocks(
			withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(1)),
			withGit(gitThatCommitsAhead()),
		)
		err := runner.RunLocal(
			project.ForProjectInput(project.Any()),
			config.Any().WithStartingBranch("main").WithExtraIterations(0),
		)
		require.Error(t, err)
		require.False(t, gitCheckoutBranchCalled(runner))
	})

	t.Run("nothing to submit", func(t *testing.T) {
		runner := withMocks(
			withProject(project.ThatReportsAllComplete()),
		)
		err := runner.RunLocal(
			project.ForProjectInput(project.Any()),
			config.Any().WithStartingBranch("main"),
		)
		require.NoError(t, err)
		require.False(t, githubPRCreated(runner))
		require.False(t, gitCheckoutBranchCalled(runner), "the checkout stays on the project branch when there is nothing to submit")
	})
}

func TestRunLocalInWorktreeDoesNotRestoreStartingCheckout(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsAllComplete()),
		withGit(gitThatCommitsAhead()),
	)
	err := runner.RunLocalInWorktree(
		project.ForProjectInput(project.Any()),
		config.Any().WithStartingBranch("main"),
	)
	require.NoError(t, err)
	require.True(t, githubPRCreated(runner))
	require.False(t, gitCheckoutBranchCalled(runner), "worktree mode leaves the starting checkout untouched")
}

func TestPrepareSetupRecordsStartingBranch(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithGit(gitOnBranch("feature-x")),
		cmdWithProject(projectWithSlug("my-project")),
	)
	setup, err := cmd.prepareSetup(flagsAny(), project.ForProjectInput(&project.Project{Slug: "my-project"}))
	require.NoError(t, err)
	require.Equal(t, "feature-x", setup.Config.StartingBranch)
}
