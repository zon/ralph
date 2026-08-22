package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

// pickerScenarioCommitLog records items 0 and 2 complete on the first query and
// every item complete on later queries, so a run offers only the remaining
// items to the picker once and then ends.
type pickerScenarioCommitLog struct {
	calls int
}

func (s *pickerScenarioCommitLog) CurrentBranch() (string, error) {
	return "csv-export", nil
}

func (s *pickerScenarioCommitLog) CommitMessages(base string) ([]string, error) {
	s.calls++
	if s.calls == 1 {
		return []string{
			"feat: first\n\ncsv-export-0",
			"feat: second\n\ncsv-export-2",
		}, nil
	}
	return []string{
		"feat: finished\n\ncsv-export-0\ncsv-export-1\ncsv-export-2\ncsv-export-3",
	}, nil
}

func TestPickerScenario_ChoosesFromIncompleteItemsOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proj.yaml")
	require.NoError(t, os.WriteFile(path, []byte(
		"- slug: one\n  description: first\n"+
			"- slug: exporter\n  description: export endpoint\n"+
			"- slug: two\n  description: second\n"+
			"- slug: importer\n  description: import endpoint\n",
	), 0o644))

	client := project.NewClient(&pickerScenarioCommitLog{}, &scenarioWarnings{})

	proj := project.WithItems(4)
	proj.Path = path

	runner := withMocks(
		withProject(client),
	)
	err := runner.RunLocal(project.ForProjectInput(proj), config.WithBase("main"))
	require.NoError(t, err)

	items := aiLastPickerItems(runner)
	require.Len(t, items, 2, "the picker is given only the remaining items")
	require.Equal(t, []int{1, 3}, itemIndices(items), "each remaining item carries its 0-based index")
	require.Equal(t, []string{"exporter", "importer"}, []string{items[0].Key(), items[1].Key()}, "each remaining item carries its key")
}
