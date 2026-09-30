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
	const report = "feat: add serializer\n\ncsv-export-IYAWN02"
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

func TestCommitIterationBlockedWithoutReportCommitsBlockedFile(t *testing.T) {
	const blocked = "blocked: cannot reach the upstream API\n\nTried the documented endpoint and a fallback."
	gitMock := gitWithChangesButNoReport()
	gitMock.blockedMessage = blocked
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete()),
		withGit(gitMock),
		withAI(&mockAI{
			runDeveloperFunc: func(_ *project.Project, _ project.Item, _ error) error {
				gitMock.blockedFile = true
				return nil
			},
		}),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.ErrorIs(t, err, ErrBlocked)
	require.True(t, gitCommittedFromBlocked(runner), "the blocked iteration is committed from blocked.md")
	require.Equal(t, blocked, gitLastCommitMessage(runner), "the commit message is the content of blocked.md")
	require.Zero(t, aiChangelogCalls(runner), "a blocked iteration never invokes the changelog agent")
	require.True(t, gitBlockedFileExists(runner), "blocked.md remains in the working tree")
}

func TestCommitIterationBlockedWithReportUsesReportMessage(t *testing.T) {
	const report = "feat: partial work\n\ntest-project-IYAWN02"
	gitMock := gitWithReport(report)
	gitMock.blockedMessage = "blocked: stopped after the report was written"
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete()),
		withGit(gitMock),
		withAI(&mockAI{
			runDeveloperFunc: func(_ *project.Project, _ project.Item, _ error) error {
				gitMock.blockedFile = true
				return nil
			},
		}),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.ErrorIs(t, err, ErrBlocked)
	require.True(t, gitCommittedFromReport(runner), "report.md stays the commit message")
	require.False(t, gitCommittedFromBlocked(runner), "blocked.md does not replace the report message")
	require.Equal(t, report, gitLastCommitMessage(runner))
	require.Zero(t, aiChangelogCalls(runner))
	require.True(t, gitBlockedFileExists(runner), "blocked.md remains in the working tree")
}

func TestCommitIterationBlockedWithNoOtherChangesCommitsBlockedFile(t *testing.T) {
	const blocked = "blocked: no code path available"
	gitMock := &mockGit{hasChanges: false, reportExists: false, blockedMessage: blocked}
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete()),
		withGit(gitMock),
		withAI(&mockAI{
			runDeveloperFunc: func(_ *project.Project, _ project.Item, _ error) error {
				gitMock.blockedFile = true
				return nil
			},
		}),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.ErrorIs(t, err, ErrBlocked)
	require.True(t, gitCommittedFromBlocked(runner), "a commit holding only blocked.md is created")
	require.Equal(t, blocked, gitLastCommitMessage(runner))
	require.Zero(t, aiChangelogCalls(runner))
}

func TestCommitIterationCreatesEmptyCommitWhenNoChangesButReport(t *testing.T) {
	const report = "feat: no code needed\n\nempty-commit-no-code-0"
	runner := withMocks(
		withProject(project.ThatReportsIncompleteUntil(1)),
		withGit(gitWithReportNoChanges(report)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.True(t, gitCommittedFromReport(runner), "an empty commit is created from the report when the working tree has no changes")
	require.Equal(t, report, gitLastCommitMessage(runner), "the report is used verbatim as the commit message")
	require.Zero(t, aiChangelogCalls(runner), "no changelog is generated while the report exists")
}
