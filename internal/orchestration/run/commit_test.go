package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

func TestCommitIterationUsesReportWhenPresent(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(1)),
		withGit(gitWithChangesAndReport()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.Zero(t, aiChangelogCalls(runner))
	require.True(t, gitCommittedFromReport(runner))
}

func TestCommitIterationDoesNotAlterReportContents(t *testing.T) {
	const report = "feat: add serializer\n\ncsv-export-0"
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(1)),
		withGit(gitWithReport(report)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.Equal(t, report, gitLastCommitMessage(runner))
}

func TestCommitIterationGeneratesChangelogWhenNoReport(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(1)),
		withGit(gitWithChangesButNoReport()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.Equal(t, 1, aiChangelogCalls(runner))
	require.True(t, gitCommittedFromReport(runner))
}

func TestCommitIterationSkipsCommitWhenNoChanges(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(1)),
		withGit(gitWithNoChanges()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.False(t, gitCommittedFromReport(runner))
}
