package setup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunSkipsPreparationWhenNoNamespaceTargeted(t *testing.T) {
	cmd := setup.withMocks(
		setup.withContext(ctx.thatTargetsNoNamespace()),
	)
	err := cmd.Run(flags.withoutNamespace())
	require.NoError(t, err)
	require.True(t, readiness.confirmGitCalled())
	require.True(t, readiness.confirmGitHubCLICalled())
	require.True(t, readiness.confirmOpenCodeCalled())
	require.False(t, github.validateCalled())
	require.False(t, github.configureCalled())
	require.False(t, github.configureTokenCalled())
	require.False(t, github.secretExistsCalled())
	require.False(t, opencode.configureCalled())
}

func TestRunReturnsErrorWhenNamespaceResolveFails(t *testing.T) {
	cmd := setup.withMocks(
		setup.withContext(ctx.thatFails()),
	)
	err := cmd.Run(flags.withoutNamespace())
	require.Error(t, err)
	require.False(t, github.validateCalled())
	require.False(t, github.configureCalled())
	require.False(t, opencode.configureCalled())
}
