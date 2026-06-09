package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestCommitIterationUsesReportWhenPresent(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(1)),
		withGit(newGitWithChangesAndReport()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.NoError(t, err)
	require.Empty(t, aiChangelogCalls(runner))
	require.True(t, gitCommittedFromReport(runner))
}

func TestCommitIterationGeneratesChangelogWhenNoReport(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(1)),
		withGit(newGitWithChangesButNoReport()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.NoError(t, err)
	require.Len(t, aiChangelogCalls(runner), 1)
	require.True(t, gitCommittedFromReport(runner))
}

func TestCommitIterationSkipsCommitWhenNoChanges(t *testing.T) {
	runner := withMocks(
		withProject(newProjectThatReportsPassingAfterIterations(1)),
		withGit(newGitWithNoChanges()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithFailingRequirements()), config.Any())
	require.NoError(t, err)
	require.False(t, gitCommittedFromReport(runner))
}
