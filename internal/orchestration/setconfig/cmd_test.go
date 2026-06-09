package setconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunConfiguresGitHubAndOpenCode(t *testing.T) {
	cmd := setconfig.withMocks()
	err := cmd.Run(flags.withKey())
	require.NoError(t, err)
	require.True(t, github.validateCalled())
	require.True(t, github.configureCalled())
	require.True(t, opencode.configureCalled())
}

func TestRunHaltsOnGitHubValidationFailure(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withGitHub(github.thatFailsValidation()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
	require.False(t, opencode.configureCalled())
}

func TestRunHaltsOnOpenCodeFailure(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withOpenCode(opencode.thatFails()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
}

func TestRunReusesExistingSecretWhenNoKeyProvided(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withGitHub(github.withExistingSecret()),
	)
	err := cmd.Run(flags.withoutKey())
	require.NoError(t, err)
	require.False(t, github.validateCalled())
	require.False(t, github.configureCalled())
}

func TestRunErrorsWhenNoKeyAndNoExistingSecret(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withGitHub(github.withNoExistingSecret()),
	)
	err := cmd.Run(flags.withoutKey())
	require.ErrorIs(t, err, ErrNoGitHubKey)
}

func TestRunPropagatesContextResolutionFailure(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withContext(ctx.thatFails()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
	require.False(t, github.validateCalled())
}

func TestRunHaltsOnSecretExistsError(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withGitHub(github.thatFailsSecretExists()),
	)
	err := cmd.Run(flags.withoutKey())
	require.Error(t, err)
	require.False(t, github.validateCalled())
}

func TestRunHaltsOnGitHubConfigureFailure(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withGitHub(github.thatFailsConfigure()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
}
