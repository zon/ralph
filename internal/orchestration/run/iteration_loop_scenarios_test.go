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

// scenarioCommitLog returns an out-of-range completion trailer on the first
// query and every item complete on later queries, so a run can continue past
// a trailer that names no item.
type scenarioCommitLog struct {
	calls int
}

func (s *scenarioCommitLog) CommitMessages(base string) ([]string, error) {
	s.calls++
	if s.calls == 1 {
		return []string{"feat: first iteration\n\ncsv-export-5"}, nil
	}
	return []string{
		"feat: finished\n\ncsv-export-0\ncsv-export-1\ncsv-export-2",
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
	err := runner.RunLocal(project.ForProjectInput(proj), config.Any())
	require.NoError(t, err)
	require.Contains(t, out.String(), "outside the resolved item array")
	require.Contains(t, out.String(), "5")
	require.Equal(t, 1, aiPickCalls(runner), "the run continues with the remaining items")
}

func TestIterationLoopScenario_DefaultExtraIterationsRoundsUp(t *testing.T) {
	runner := withMocks(
		withProject(project.ThatAlwaysReportsIncomplete().WithResolvedItems(3)),
	)
	err := runner.RunLocal(project.ForProjectInput(project.WithItems(3)), config.Any())
	require.Error(t, err)
	require.Equal(t, 4, aiPickCalls(runner))
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
