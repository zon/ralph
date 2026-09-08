package setup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunConfirmsLocalReadinessBeforeClusterWork(t *testing.T) {
	cmd := setup.withMocks()
	err := cmd.Run(flags.withKey())
	require.NoError(t, err)
	require.True(t, readiness.confirmGitCalled())
	require.True(t, readiness.confirmGitHubCLICalled())
	require.True(t, readiness.confirmOpenCodeCalled())
	require.True(t, ctx.resolveCalled())
}

func TestRunHaltsBeforeGitHubCLIWhenGitNotReady(t *testing.T) {
	cmd := setup.withMocks(
		setup.withReadiness(readiness.thatFailsGit()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
	require.False(t, readiness.confirmGitHubCLICalled())
	require.False(t, readiness.confirmOpenCodeCalled())
	require.False(t, ctx.resolveCalled())
	require.False(t, github.validateCalled())
	require.False(t, opencode.configureCalled())
}

func TestRunHaltsBeforeOpenCodeWhenGitHubCLINotReady(t *testing.T) {
	cmd := setup.withMocks(
		setup.withReadiness(readiness.thatFailsGitHubCLI()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
	require.True(t, readiness.confirmGitCalled())
	require.False(t, readiness.confirmOpenCodeCalled())
	require.False(t, ctx.resolveCalled())
}

func TestRunHaltsBeforeClusterWorkWhenOpenCodeNotReady(t *testing.T) {
	cmd := setup.withMocks(
		setup.withReadiness(readiness.thatFailsOpenCode()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
	require.True(t, readiness.confirmGitCalled())
	require.True(t, readiness.confirmGitHubCLICalled())
	require.False(t, ctx.resolveCalled())
	require.False(t, github.validateCalled())
	require.False(t, opencode.configureCalled())
}
