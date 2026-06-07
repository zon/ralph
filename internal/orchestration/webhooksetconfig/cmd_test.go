package webhooksetconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunWritesConfigThenSecrets(t *testing.T) {
	cmd := webhooksetconfig.withMocks()
	err := cmd.Run(flags.any())

	require.NoError(t, err)
	require.True(t, config.writeCalled())
	require.True(t, secrets.writeCalled())
}

func TestRunHaltsOnConfigWriteFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withConfig(config.thatFailsWrite()),
	)
	err := cmd.Run(flags.any())

	require.Error(t, err)
	require.False(t, secrets.generateCalled())
	require.False(t, secrets.writeCalled())
}

func TestRunContinuesAfterWebhookRegistrationFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withGitHub(github.thatFailsRegistration()),
	)
	err := cmd.Run(flags.any())

	require.NoError(t, err)
	require.True(t, secrets.writeCalled())
}

func TestRunPropagatesContextResolutionFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withContext(ctx.thatFails()),
	)
	err := cmd.Run(flags.any())

	require.Error(t, err)
	require.False(t, config.writeCalled())
}
