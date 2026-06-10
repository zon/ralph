package workflowrun

import (
	"testing"

	"github.com/stretchr/testify/require"
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
		run.withProject(project.thatFailsLoad()),
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
