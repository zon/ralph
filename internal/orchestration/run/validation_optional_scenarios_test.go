package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

// realResolvingProjectClient wraps the real project client so a run resolves
// the project file with the production parse and query implementation while the
// test records resolution and reports every item complete so the run exits.
type realResolvingProjectClient struct {
	*project.Client
	resolveCount int
	lastQuery    string
	lastPath     string
	lastProject  *project.Project
}

func (c *realResolvingProjectClient) Resolve(path, query string) (*project.Project, error) {
	c.resolveCount++
	c.lastQuery = query
	c.lastPath = path
	proj, err := c.Client.Resolve(path, query)
	if err != nil {
		return nil, err
	}
	c.lastProject = proj
	return proj, nil
}

func (c *realResolvingProjectClient) Incomplete(proj *project.Project, base string) ([]project.Item, error) {
	return nil, nil
}

type emptyCommitLog struct{}

func (emptyCommitLog) CurrentBranch() (string, error) {
	return "csv-export", nil
}

func (emptyCommitLog) CommitMessages(base string) ([]string, error) {
	return nil, nil
}

type warnNop struct{}

func (warnNop) Warnf(format string, a ...any) {}

func writeForeignFile(t *testing.T) string {
	t.Helper()
	content := "jobs:\n  - name: build\n    steps: [checkout, test]\n  - name: deploy\n    steps: [build, push]\n"
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// TestRunLocalForeignFileNotReformatted covers the "Foreign project file run
// without validation" scenario: a project file owned by another tool, whose
// formatting must not be rewritten, resolves items normally when ralph runs
// against it without validating first, and the file is left untouched.
func TestRunLocalForeignFileNotReformatted(t *testing.T) {
	// GIVEN a project file owned by another tool, whose formatting must not be
	// rewritten
	content := "jobs:\n  - name: build\n    steps: [checkout, test]\n  - name: deploy\n    steps: [build, push]\n"
	path := filepath.Join(t.TempDir(), "workflow.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	client := &realResolvingProjectClient{Client: project.NewClient(&emptyCommitLog{}, &warnNop{})}
	runner := withMocks(
		withProject(client),
	)

	// WHEN the user runs ralph against it without validating first
	err := runner.RunLocal(project.ForProjectInput(&project.Project{Path: path}), config.WithItems(".jobs"))
	require.NoError(t, err)

	// THEN the run resolves items normally
	require.Equal(t, 1, client.resolveCount)
	require.Equal(t, ".jobs", client.lastQuery)
	require.NotNil(t, client.lastProject)
	require.Len(t, client.lastProject.Items, 2)

	// AND the file is not reformatted
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(after))
}

// TestRunLocalNeverValidatedFileRunnable covers the item that validation is a
// convenience and not a precondition: a project file that has never been
// validated is runnable.
func TestRunLocalNeverValidatedFileRunnable(t *testing.T) {
	path := writeForeignFile(t)
	client := &realResolvingProjectClient{Client: project.NewClient(&emptyCommitLog{}, &warnNop{})}
	runner := withMocks(
		withProject(client),
	)

	err := runner.RunLocal(project.ForProjectInput(&project.Project{Path: path}), config.WithItems(".jobs"))
	require.NoError(t, err)
	require.Equal(t, 1, client.resolveCount)
}

// TestRunLocalReportsSameQueryErrorsAsValidate covers the item that a run
// reports the same query errors validation does: a query that yields no items
// and a query that cannot be evaluated surface identically through the run.
func TestRunLocalReportsSameQueryErrorsAsValidate(t *testing.T) {
	path := writeForeignFile(t)

	t.Run("query yielding no items", func(t *testing.T) {
		client := &realResolvingProjectClient{Client: project.NewClient(&emptyCommitLog{}, &warnNop{})}
		runner := withMocks(withProject(client))

		err := runner.RunLocal(project.ForProjectInput(&project.Project{Path: path}), config.WithItems(".missing"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item query yielded no items: .missing")
	})

	t.Run("query that cannot be evaluated", func(t *testing.T) {
		client := &realResolvingProjectClient{Client: project.NewClient(&emptyCommitLog{}, &warnNop{})}
		runner := withMocks(withProject(client))

		err := runner.RunLocal(project.ForProjectInput(&project.Project{Path: path}), config.WithItems(".jobs[].name.deep"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item query failed")
		assert.Contains(t, err.Error(), ".jobs[].name.deep")
	})
}
