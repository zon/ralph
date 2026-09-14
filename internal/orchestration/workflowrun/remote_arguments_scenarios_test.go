package workflowrun

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ralphcfg "github.com/zon/ralph/internal/config"
	ralphproj "github.com/zon/ralph/internal/project"
)

// TestWorkflowRunConfigChangeAfterSubmissionScenario covers the "Config change
// after submission does not affect the run" scenario: a workflow submitted
// carrying `--items .requirements` resolves items with `.requirements` even when
// `.ralph/config.yaml` was changed on the branch after submission.
func TestWorkflowRunConfigChangeAfterSubmissionScenario(t *testing.T) {
	// GIVEN a workflow has been submitted carrying `--items .requirements`
	// AND `.ralph/config.yaml` is later changed on the branch
	cfg := ralphcfg.Any()
	cfg.Items = ".changed-after-submission"
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

	// WHEN the container runs
	err := cmd.Run(flags.withItems(".requirements"))

	// THEN it resolves items with `.requirements` as submitted
	require.NoError(t, err)
	require.Equal(t, ".requirements", projMock.lastQuery)
	assert.NotEqual(t, configItems, projMock.lastQuery)
	require.NotNil(t, capturedCfg)
	assert.Equal(t, ".requirements", capturedCfg.Items)
}

// TestWorkflowRunContainerSynchronizesDeliveredBaseBranch covers the "Container
// synchronizes the base branch" scenario: the base branch delivered to the
// container as `--base` is applied to the run-local config so the delegated
// run-local behavior fetches and merges exactly that branch before the first
// iteration.
func TestWorkflowRunContainerSynchronizesDeliveredBaseBranch(t *testing.T) {
	cfg := ralphcfg.Any()
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
		run.withRunner(runnerMock),
	)

	err := cmd.Run(flags.withBaseBranch("develop"))

	require.NoError(t, err)
	require.True(t, runner.runLocalCalled(), "the container delegates to the run-local behavior that synchronizes the base branch")
	require.NotNil(t, capturedCfg)
	assert.Equal(t, "develop", capturedCfg.Base, "synchronization uses the base branch delivered as --base")
}

// TestWorkflowRunContainerDoesNotReResolveItemQuery covers the item that the
// container never re-resolves the item query from the repository config: the
// submitted `--items` value wins over the config `items` field.
func TestWorkflowRunContainerDoesNotReResolveItemQuery(t *testing.T) {
	cfg := ralphcfg.Any()
	cfg.Items = ".config-query"
	projMock := &mockProjectClient{}
	cmd := run.withMocks(
		run.withConfig(&mockConfigClient{
			loadOptionalFunc: func() (*ralphcfg.RalphConfig, error) { return cfg, nil },
		}),
		run.withProject(projMock),
	)

	err := cmd.Run(flags.withItems(".requirements"))

	require.NoError(t, err)
	require.Equal(t, ".requirements", projMock.lastQuery)
}
