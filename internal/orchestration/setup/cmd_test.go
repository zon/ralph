package setup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunConfiguresGitHubAndOpenCode(t *testing.T) {
	cmd := setup.withMocks()
	_, err := cmd.Run(flags.withKey())
	require.NoError(t, err)
	require.True(t, github.validateCalled())
	require.True(t, github.configureCalled())
	require.True(t, opencode.configureCalled())
}

func TestRunHaltsOnGitHubValidationFailure(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(github.thatFailsValidation()),
	)
	_, err := cmd.Run(flags.withKey())
	require.Error(t, err)
	require.False(t, opencode.configureCalled())
}

func TestRunHaltsOnOpenCodeFailure(t *testing.T) {
	cmd := setup.withMocks(
		setup.withOpenCode(opencode.thatFails()),
	)
	_, err := cmd.Run(flags.withKey())
	require.Error(t, err)
}

func TestRunReusesExistingSecretWhenNoKeyProvided(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(github.withExistingSecret()),
	)
	result, err := cmd.Run(flags.withoutKey())
	require.NoError(t, err)
	require.True(t, result.GitHubSecretReused)
	require.False(t, github.validateCalled())
	require.False(t, github.configureCalled())
}

func TestRunErrorsWhenNoKeyAndNoExistingSecret(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(github.withNoExistingSecret()),
	)
	_, err := cmd.Run(flags.withoutKey())
	require.ErrorIs(t, err, ErrNoGitHubKey)
	require.True(t, github.ghCliTokenCalled())
	require.True(t, github.envTokenCalled())
	require.False(t, github.configureTokenCalled())
	require.False(t, github.configureCalled())
	require.False(t, opencode.configureCalled())
}

func TestRunPropagatesContextResolutionFailure(t *testing.T) {
	cmd := setup.withMocks(
		setup.withContext(ctx.thatFails()),
	)
	_, err := cmd.Run(flags.withKey())
	require.Error(t, err)
	require.False(t, github.validateCalled())
}

func TestRunHaltsOnSecretExistsError(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(github.thatFailsSecretExists()),
	)
	_, err := cmd.Run(flags.withoutKey())
	require.Error(t, err)
	require.False(t, github.validateCalled())
}

func TestRunHaltsOnGitHubConfigureFailure(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(github.thatFailsConfigure()),
	)
	_, err := cmd.Run(flags.withKey())
	require.Error(t, err)
}

func TestRunConfiguresGitHubTokenWithoutValidation(t *testing.T) {
	cmd := setup.withMocks()
	_, err := cmd.Run(flags.withToken())
	require.NoError(t, err)
	require.True(t, github.configureTokenCalled())
	require.False(t, github.validateCalled())
	require.False(t, github.configureCalled())
	require.True(t, opencode.configureCalled())
}

func TestRunHaltsOnGitHubTokenConfigureFailure(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(github.thatFailsConfigureToken()),
	)
	_, err := cmd.Run(flags.withToken())
	require.Error(t, err)
	require.False(t, opencode.configureCalled())
}

func TestRunErrorsWhenBothKeyAndTokenProvided(t *testing.T) {
	cmd := setup.withMocks()
	_, err := cmd.Run(flags.withKeyAndToken())
	require.ErrorIs(t, err, ErrBothGitHubFlags)
	require.False(t, ctx.resolveCalled())
	require.False(t, github.validateCalled())
	require.False(t, github.configureCalled())
	require.False(t, github.configureTokenCalled())
	require.False(t, opencode.configureCalled())
}

func TestRunFallsBackToGHCliTokenWhenNoSecret(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(github.withGHCliToken("ghp_cli_token")),
	)
	_, err := cmd.Run(flags.withoutKey())
	require.NoError(t, err)
	require.True(t, github.configureTokenCalled())
	require.Equal(t, "ghp_cli_token", github.configuredToken())
	require.False(t, github.envTokenCalled())
	require.True(t, opencode.configureCalled())
}

func TestRunFallsBackToEnvTokenWhenNoSecretAndNoGHCliToken(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(github.withEnvToken("ghp_env_token")),
	)
	_, err := cmd.Run(flags.withoutKey())
	require.NoError(t, err)
	require.True(t, github.configureTokenCalled())
	require.Equal(t, "ghp_env_token", github.configuredToken())
	require.True(t, github.ghCliTokenCalled())
	require.True(t, opencode.configureCalled())
}

func TestRunChecksWorkflowScopeOfReusedToken(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(&mockGitHubCredentialsClient{
			secretExistsFunc: func(K8sContext) (bool, error) { return true, nil },
			tokenFromSecretFunc: func(K8sContext) (string, error) {
				return "gho_reused", nil
			},
			tokenWorkflowScopeFunc: func(string) (bool, bool, error) {
				return false, true, nil
			},
		}),
	)

	result, err := cmd.Run(flags.withoutKey())

	require.NoError(t, err)
	require.True(t, result.GitHubSecretReused)
	require.True(t, github.tokenFromSecretCalled())
	require.True(t, github.workflowScopeCalled())
	require.Equal(t, "gho_reused", github.scopedToken())
	require.True(t, result.WorkflowScopeKnown)
	require.False(t, result.WorkflowScopePresent)
}

func TestRunChecksWorkflowScopeOfProvidedToken(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(&mockGitHubCredentialsClient{
			tokenWorkflowScopeFunc: func(string) (bool, bool, error) {
				return true, true, nil
			},
		}),
	)

	result, err := cmd.Run(flags.withToken())

	require.NoError(t, err)
	require.False(t, result.GitHubSecretReused)
	require.True(t, github.workflowScopeCalled())
	require.True(t, result.WorkflowScopeKnown)
	require.True(t, result.WorkflowScopePresent)
}

func TestRunSkipsWorkflowScopeCheckForAppCredentials(t *testing.T) {
	cmd := setup.withMocks()

	_, err := cmd.Run(flags.withKey())

	require.NoError(t, err)
	require.False(t, github.workflowScopeCalled())
}

func TestRunLeavesWorkflowScopeUnknownWhenCheckFails(t *testing.T) {
	cmd := setup.withMocks(
		setup.withGitHub(&mockGitHubCredentialsClient{
			tokenWorkflowScopeFunc: func(string) (bool, bool, error) {
				return false, false, errMock
			},
		}),
	)

	result, err := cmd.Run(flags.withToken())

	require.NoError(t, err)
	require.False(t, result.WorkflowScopeKnown)
	require.False(t, result.WorkflowScopePresent)
	require.True(t, opencode.configureCalled())
}
