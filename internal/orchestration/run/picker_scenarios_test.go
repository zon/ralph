package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

// pickerScenarioItems are the four items the picker scenario project resolves
// to, shared between the project file the run reads and the commit log that
// records completion, so the trailer hashes match the resolved items.
var pickerScenarioItems = []any{
	map[string]any{"slug": "one", "description": "first"},
	map[string]any{"slug": "exporter", "description": "export endpoint"},
	map[string]any{"slug": "two", "description": "second"},
	map[string]any{"slug": "importer", "description": "import endpoint"},
}

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
	items := project.NewItems(pickerScenarioItems)
	s.calls++
	if s.calls == 1 {
		return []string{
			"feat: first\n\ncsv-export-" + items[0].Hash(),
			"feat: second\n\ncsv-export-" + items[2].Hash(),
		}, nil
	}
	return []string{
		"feat: finished\n\ncsv-export-" + items[0].Hash() + "\ncsv-export-" + items[1].Hash() + "\ncsv-export-" + items[2].Hash() + "\ncsv-export-" + items[3].Hash(),
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
