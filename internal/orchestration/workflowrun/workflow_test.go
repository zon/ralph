package workflowrun

import (
	"testing"

	"github.com/stretchr/testify/require"

	ralphcfg "github.com/zon/ralph/internal/config"
	ralphproj "github.com/zon/ralph/internal/project"
)

func TestRunMissingProjectPathAbortsBeforeWorkspace(t *testing.T) {
	cmd := run.withMocks()
	err := cmd.Run(flags.withNoProjectPath())
	require.Error(t, err)
	require.False(t, workspace.setupCalled())
}

func TestRunWorkspaceFailureAbortsEarly(t *testing.T) {
	cmd := run.withMocks(
		run.withWorkspace(workspace.thatFailsSetup()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, config.loadCalled())
}

func TestRunDebugSetupFailureAbortsEarly(t *testing.T) {
	cmd := run.withMocks(
		run.withDebug(debug.thatFailsSetup()),
	)
	err := cmd.Run(flags.withDebugBranch("my-ralph-branch"))
	require.Error(t, err)
	require.False(t, config.loadCalled())
}

func TestRunMissingConfigProceedsWithDefaults(t *testing.T) {
	cmd := run.withMocks(
		run.withConfig(config.thatReportsMissing()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, runner.runLocalCalled())
}

func TestRunMalformedConfigAbortsBeforeSync(t *testing.T) {
	cmd := run.withMocks(
		run.withConfig(config.thatFailsParsing()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, git.fetchCalled())
}

func TestRunProjectLoadFailureAbortsBeforeSync(t *testing.T) {
	cmd := run.withMocks(
		run.withProject(project.thatFailsResolve()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, git.fetchCalled())
}

func TestSyncBaseBranchFetchFailureContinues(t *testing.T) {
	cmd := run.withMocks(
		run.withGit(git.thatFailsFetch()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, runner.runLocalCalled())
	require.True(t, output.warnfCalled())
}

func TestSyncBaseBranchUpToDateSkipsMerge(t *testing.T) {
	cmd := run.withMocks(
		run.withGit(git.thatReportsUpToDate()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.False(t, git.mergeCalled())
}

func TestSyncBaseBranchConflictsAbortAndInvokeAI(t *testing.T) {
	cmd := run.withMocks(
		run.withGit(git.thatNeedsMerge().thatProducesConflicts()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, git.mergeAborted())
	require.True(t, ai.conflictsResolved())
}

func TestRunDelegatesToLocalRunner(t *testing.T) {
	cmd := run.withMocks()
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, runner.runLocalCalled())
}

func TestRunExtraIterationsAppliedToConfig(t *testing.T) {
	var capturedCfg *ralphcfg.RalphConfig
	mockRunner := &mockRunnerClient{
		runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
			capturedCfg = cfg
			return nil
		},
	}
	cmd := run.withMocks(
		run.withRunner(mockRunner),
	)
	err := cmd.Run(flags.withExtraIterations(2))
	require.NoError(t, err)
	require.NotNil(t, capturedCfg)
	require.NotNil(t, capturedCfg.ExtraIterations)
	require.Equal(t, 2, *capturedCfg.ExtraIterations)
}

func TestRunExtraIterationsAbsentDefaultsToConfig(t *testing.T) {
	var capturedCfg *ralphcfg.RalphConfig
	mockRunner := &mockRunnerClient{
		runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
			capturedCfg = cfg
			return nil
		},
	}
	cmd := run.withMocks(
		run.withRunner(mockRunner),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.NotNil(t, capturedCfg)
	require.Nil(t, capturedCfg.ExtraIterations)
}

func TestRunNoServicesClearsConfigServices(t *testing.T) {
	cfg := ralphcfg.Any()
	cfg.Services = []ralphcfg.Service{
		{Name: "test-svc", Command: "echo"},
	}
	mockCfg := &mockConfigClient{
		loadOptionalFunc: func() (*ralphcfg.RalphConfig, error) { return cfg, nil },
	}
	var capturedCfg *ralphcfg.RalphConfig
	mockRunner := &mockRunnerClient{
		runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
			capturedCfg = cfg
			return nil
		},
	}
	cf := flags.any()
	cf.NoServices = true
	cmd := run.withMocks(
		run.withConfig(mockCfg),
		run.withRunner(mockRunner),
	)
	err := cmd.Run(cf)
	require.NoError(t, err)
	require.NotNil(t, capturedCfg)
	require.Nil(t, capturedCfg.Services)
}

func TestRunBaseBranchBoundsCompletionLog(t *testing.T) {
	var capturedCfg *ralphcfg.RalphConfig
	mockRunner := &mockRunnerClient{
		runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
			capturedCfg = cfg
			return nil
		},
	}
	cf := flags.any()
	cf.BaseBranch = "develop"
	cmd := run.withMocks(
		run.withRunner(mockRunner),
	)
	err := cmd.Run(cf)
	require.NoError(t, err)
	require.NotNil(t, capturedCfg)
	require.Equal(t, "develop", capturedCfg.Base, "the base branch bounds the commit log completion is read from")
}

func TestRunItemsAppliedToConfig(t *testing.T) {
	var capturedCfg *ralphcfg.RalphConfig
	mockRunner := &mockRunnerClient{
		runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
			capturedCfg = cfg
			return nil
		},
	}
	cmd := run.withMocks(
		run.withRunner(mockRunner),
	)
	err := cmd.Run(flags.withItems(".spec.tasks"))
	require.NoError(t, err)
	require.NotNil(t, capturedCfg)
	require.Equal(t, ".spec.tasks", capturedCfg.Items)
}

func TestRunItemsAbsentLeavesConfigQuery(t *testing.T) {
	var capturedCfg *ralphcfg.RalphConfig
	mockRunner := &mockRunnerClient{
		runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
			capturedCfg = cfg
			return nil
		},
	}
	cmd := run.withMocks(
		run.withRunner(mockRunner),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.NotNil(t, capturedCfg)
	require.Equal(t, ".", capturedCfg.Items)
}

func TestRunCleanupAppliedToConfig(t *testing.T) {
	var capturedCfg *ralphcfg.RalphConfig
	mockRunner := &mockRunnerClient{
		runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
			capturedCfg = cfg
			return nil
		},
	}
	cmd := run.withMocks(
		run.withRunner(mockRunner),
	)
	err := cmd.Run(flags.withCleanup())
	require.NoError(t, err)
	require.NotNil(t, capturedCfg)
	require.True(t, capturedCfg.Cleanup)
}

func TestRunCleanupAbsentLeavesCleanupDisabled(t *testing.T) {
	var capturedCfg *ralphcfg.RalphConfig
	mockRunner := &mockRunnerClient{
		runLocalFunc: func(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
			capturedCfg = cfg
			return nil
		},
	}
	cmd := run.withMocks(
		run.withRunner(mockRunner),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.NotNil(t, capturedCfg)
	require.False(t, capturedCfg.Cleanup)
}
