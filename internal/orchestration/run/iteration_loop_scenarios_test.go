package run

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

// scenarioCommitLog returns a completion trailer naming no resolved item on the
// first query and every item complete on later queries, so a run can continue
// past a trailer that matches no item.
type scenarioCommitLog struct {
	calls int
}

func (s *scenarioCommitLog) CurrentBranch() (string, error) {
	return "csv-export", nil
}

func (s *scenarioCommitLog) CommitMessages(base string) ([]string, error) {
	s.calls++
	if s.calls == 1 {
		return []string{"feat: first iteration\n\ncsv-export-" + project.NewItems([]any{"four"})[0].Hash()}, nil
	}
	items := project.NewItems([]any{"one", "two", "three"})
	return []string{
		"feat: finished\n\ncsv-export-" + items[0].Hash() + "\ncsv-export-" + items[1].Hash() + "\ncsv-export-" + items[2].Hash(),
	}, nil
}

// scenarioWarnings captures the warnings emitted during completion
// reconciliation so a test can assert them.
type scenarioWarnings struct {
	strings.Builder
}

func (s *scenarioWarnings) Warnf(format string, a ...any) {
	fmt.Fprintf(s, format, a...)
}

func TestIterationLoopScenario_InterruptedRunResumesFromTheLog(t *testing.T) {
	projMock := project.ThatReportsComplete(0, 1).WithResolvedItems(3).ThenAllComplete()
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithBase("main"))
	require.NoError(t, err)
	require.Equal(t, "main", projMock.LastBase(), "completion is read from the commit log bounded by the base branch")
	require.Equal(t, []int{2}, aiLastPickerIndices(runner), "the loop continues with the items that are left")
}

func TestIterationLoopScenario_OutOfRangeTrailerIgnoredWithWarning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proj.yaml")
	require.NoError(t, os.WriteFile(path, []byte("- one\n- two\n- three\n"), 0o644))

	log := &scenarioCommitLog{}
	out := &scenarioWarnings{}
	client := project.NewClient(log, out)

	proj := project.WithItems(3)
	proj.Path = path

	runner := withMocks(
		withProject(client),
	)
	err := runner.RunLocal(project.ForProjectInput(proj), config.Any().WithoutCleanup())
	require.NoError(t, err)
	require.Contains(t, out.String(), "matches no resolved item")
	require.Contains(t, out.String(), project.NewItems([]any{"four"})[0].Hash())
	require.Equal(t, 1, aiPickCalls(runner), "the run continues with the remaining items")
}

// TestIterationLoopScenario_BlockedDuringIterationStopsLoop covers the
// `blocked.md` written during an iteration scenario: the iteration is
// committed and the loop stops with a blocked error without starting another
// iteration, so the picker is not invoked again even though iterations remain
// in the limit.
func TestIterationLoopScenario_BlockedDuringIterationStopsLoop(t *testing.T) {
	gitMock := gitWithChangesButNoReport()
	gitMock.blockedMessage = "blocked: cannot reach the upstream API"
	projMock := project.ThatAlwaysReportsIncomplete().WithResolvedItems(3)
	runner := withMocks(
		withProject(projMock),
		withGit(gitMock),
		withAI(&mockAI{
			runDeveloperFunc: func(_ *project.Project, _ project.Item, _ error) error {
				gitMock.blockedFile = true
				return nil
			},
		}),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(3))
	require.ErrorIs(t, err, ErrBlocked)
	require.True(t, gitCommittedFromBlocked(runner), "the blocked iteration is committed before the loop stops")
	require.Equal(t, 1, aiPickCalls(runner), "the picker is not invoked again after the blocked iteration")
	require.Equal(t, 1, projMock.IncompleteCallCount(), "the loop does not start another iteration")
}

// TestIterationLoopScenario_BlockedOnBranchStopsResumedRun covers the
// `blocked.md` on the branch scenario: a run started against a branch that
// already carries blocked.md stops with a blocked error before invoking the
// AI.
func TestIterationLoopScenario_BlockedOnBranchStopsResumedRun(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(3)),
		withGit(gitWithBlockedFile()),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.WithExtraIterations(3))
	require.ErrorIs(t, err, ErrBlocked)
	require.Zero(t, aiPickCalls(runner), "the resumed run is stopped before the picker runs")
	require.Zero(t, aiDevelopCalls(runner), "the resumed run is stopped before the developer runs")
	require.False(t, gitCommittedFromBlocked(runner), "the resumed run commits nothing before stopping")
}

func TestIterationLoopScenario_DefaultExtraIterationsRoundsUp(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(3)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.Equal(t, 4, aiPickCalls(runner))
}

func TestIterationLoopScenario_DefaultExtraIterationsIs30PercentWhenUnset(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(10)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(10)), config.Any())
	require.Error(t, err)
	require.Equal(t, 13, aiPickCalls(runner))
}

func TestIterationLoopNeverConsultsProjectFileForCompletion(t *testing.T) {
	projMock := project.ThatReportsIncompleteUntil(2)
	runner := withMocks(
		withProject(projMock),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.NoError(t, err)
	require.Equal(t, 1, projMock.ResolveCount(), "the project file is resolved once and never re-read for completion")
	require.Equal(t, 3, projMock.IncompleteCallCount(), "completion is read from the commit log each iteration")
	require.False(t, projMock.Written(), "the project file is never written during the loop")
}
