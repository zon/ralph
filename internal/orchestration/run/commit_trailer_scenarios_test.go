package run

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/trailer"
)

func TestCommitTrailerScenario_ReportWithoutTrailerLeavesItemIncomplete(t *testing.T) {
	const report = "refactor: extract config loader\n\nNo completion trailer here"
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2)),
		withGit(gitWithReport(report)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.True(t, gitCommittedFromReport(runner), "the commit is created")
	require.Equal(t, report, gitLastCommitMessage(runner), "the report is used verbatim as the commit message")
	require.Zero(t, aiChangelogCalls(runner), "no changelog is generated while the report exists")
	require.Equal(t, 2, aiPickCalls(runner), "no item is marked complete, so the loop runs another iteration")
	require.Equal(t, []int{0, 1, 2}, aiLastPickerIndices(runner), "the picker may select the same item again in a later iteration")
}

func TestCommitTrailerScenario_NoCodeNeededRecordsCompletion(t *testing.T) {
	const report = "feat: no code needed\n\ncsv-export-7FhX6dT"
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2)),
		withGit(gitWithReportNoChanges(report)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.True(t, gitCommittedFromReport(runner), "an empty commit is created from the report")
	require.Equal(t, report, gitLastCommitMessage(runner), "the report is used verbatim as the commit message")
	require.Zero(t, aiChangelogCalls(runner), "no changelog is generated while the report exists")
}

func TestCommitTrailerScenario_ChangesWithoutReportGenerateChangelog(t *testing.T) {
	const changelog = "feat: wire the item array through run-local\n\nThe generated changelog never contains a completion trailer"
	gitMock := gitWithChangesButNoReport()
	gitMock.reportMessage = changelog
	aiMock := &mockAI{
		changelogFunc: func() error {
			gitMock.reportExists = true
			return nil
		},
	}
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(2)),
		withGit(gitMock),
		withAI(aiMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.Equal(t, 1, aiChangelogCalls(runner), "the AI is called to generate a changelog because no report.md exists")
	require.True(t, gitCommittedFromReport(runner), "the generated report.md is used as the commit message")
	require.Equal(t, changelog, gitLastCommitMessage(runner))
	require.Empty(t, trailer.Parse(changelog), "the generated changelog records no item complete")
	require.Equal(t, 2, aiPickCalls(runner), "the generated changelog records no item complete, so the loop runs another iteration")
	require.Equal(t, []int{0, 1, 2}, aiLastPickerIndices(runner), "the picker may select the same item again in a later iteration")
}
