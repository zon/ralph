package workflowrun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ralphcfg "github.com/zon/ralph/internal/config"
	ralphproj "github.com/zon/ralph/internal/project"
)

type emptyCommitLog struct{}

func (emptyCommitLog) CommitMessages(base string) ([]string, error) { return nil, nil }

type warnNop struct{}

func (warnNop) Warnf(format string, a ...any) {}

func writeTempProject(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// TestWorkflowRunItemsAbsentUsesConfigQuery covers the item that the item
// query is resolved with the run's precedence: when --items is absent, the
// config items field supplies the query used to resolve the item array.
func TestWorkflowRunItemsAbsentUsesConfigQuery(t *testing.T) {
	cfg := ralphcfg.Any()
	cfg.Items = ".requirements"
	projMock := &mockProjectClient{}
	cmd := run.withMocks(
		run.withConfig(&mockConfigClient{
			loadOptionalFunc: func() (*ralphcfg.RalphConfig, error) { return cfg, nil },
		}),
		run.withProject(projMock),
	)

	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.Equal(t, ".requirements", projMock.lastQuery)
}

// TestWorkflowRunValidationFailsBeforeSync covers the item that input
// validation after workspace setup fails before base-branch synchronization
// when the project file is missing, does not parse, or yields no items under
// the supplied item query.
func TestWorkflowRunValidationFailsBeforeSync(t *testing.T) {
	cfg := ralphcfg.Any()
	cfg.Items = ".missing"
	cmd := run.withMocks(
		run.withConfig(&mockConfigClient{
			loadOptionalFunc: func() (*ralphcfg.RalphConfig, error) { return cfg, nil },
		}),
		run.withProject(project.thatFailsResolve()),
	)

	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, git.fetchCalled())
	require.False(t, runner.runLocalCalled())
}

// TestWorkflowRunItemQueryScenario covers the "Item query" scenario: when
// --items is provided with the query resolved at submission time, that query is
// used to resolve the item array and the items field in .ralph/config.yaml is
// not consulted.
func TestWorkflowRunItemQueryScenario(t *testing.T) {
	// GIVEN --items is provided and the config carries a different items query
	cfg := ralphcfg.Any()
	cfg.Items = ".config-queries"
	configItems := cfg.Items
	projMock := &mockProjectClient{}
	var capturedCfg *ralphcfg.RalphConfig
	runnerMock := &mockRunnerClient{
		runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
			capturedCfg = cfg
			return nil
		},
	}
	cmd := run.withMocks(
		run.withConfig(&mockConfigClient{
			loadOptionalFunc: func() (*ralphcfg.RalphConfig, error) { return cfg, nil },
		}),
		run.withProject(projMock),
		run.withRunner(runnerMock),
	)

	// WHEN the project loop executes
	err := cmd.Run(flags.withItems(".spec.tasks"))

	// THEN that query is used to resolve the item array
	require.NoError(t, err)
	require.Equal(t, ".spec.tasks", projMock.lastQuery)
	require.NotEqual(t, configItems, projMock.lastQuery)
	// AND the items field in .ralph/config.yaml is not consulted: the flag
	// query wins over the config's query
	require.NotNil(t, capturedCfg)
	assert.Equal(t, ".spec.tasks", capturedCfg.Items)
}

// TestWorkflowRunCleanupScenario covers the "Cleanup" scenario at the workflow
// run boundary: the --cleanup flag is passed through to the local execution
// behavior, which deletes and commits the project file before the pull request
// when every item is complete and leaves it in place when the flag is absent.
func TestWorkflowRunCleanupScenario(t *testing.T) {
	t.Run("cleanup provided is passed through", func(t *testing.T) {
		var capturedCfg *ralphcfg.RalphConfig
		runnerMock := &mockRunnerClient{
			runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
				capturedCfg = cfg
				return nil
			},
		}
		cmd := run.withMocks(
			run.withRunner(runnerMock),
		)
		err := cmd.Run(flags.withCleanup())
		require.NoError(t, err)
		require.NotNil(t, capturedCfg)
		require.True(t, capturedCfg.Cleanup)
	})

	t.Run("cleanup absent leaves file in place", func(t *testing.T) {
		var capturedCfg *ralphcfg.RalphConfig
		runnerMock := &mockRunnerClient{
			runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
				capturedCfg = cfg
				return nil
			},
		}
		cmd := run.withMocks(
			run.withRunner(runnerMock),
		)
		err := cmd.Run(flags.any())
		require.NoError(t, err)
		require.NotNil(t, capturedCfg)
		require.False(t, capturedCfg.Cleanup)
	})
}

// TestWorkflowRunProjectFileLoadFailureScenario covers the "Project file load
// failure" scenario: after the workspace is ready, a project file that is
// missing, does not parse, or yields no items under the supplied item query
// returns an error before base-branch synchronization begins. The real project
// client resolves the file exactly as a run does.
func TestWorkflowRunProjectFileLoadFailureScenario(t *testing.T) {
	client := ralphproj.NewClient(&emptyCommitLog{}, &warnNop{})

	t.Run("missing file", func(t *testing.T) {
		cmd := run.withMocks(
			run.withProject(client),
		)
		err := cmd.Run(flags.withItems(".requirements"))
		require.Error(t, err)
		require.False(t, git.fetchCalled())
	})

	t.Run("file does not parse", func(t *testing.T) {
		path := writeTempProject(t, "broken.yaml", "slug: [unclosed\n")
		cf := flags.any()
		cf.ProjectPath = path
		cmd := run.withMocks(
			run.withProject(client),
		)
		err := cmd.Run(cf)
		require.Error(t, err)
		require.False(t, git.fetchCalled())
	})

	t.Run("query yields no items", func(t *testing.T) {
		path := writeTempProject(t, "project.yaml", "requirements: []\n")
		cf := flags.withItems(".requirements")
		cf.ProjectPath = path
		cmd := run.withMocks(
			run.withProject(client),
		)
		err := cmd.Run(cf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item query yielded no items: .requirements")
		require.False(t, git.fetchCalled())
	})
}
