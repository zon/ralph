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

func TestRunCallsWebhookRegistration(t *testing.T) {
	cmd := webhooksetconfig.withMocks()
	err := cmd.Run(flags.any())

	require.NoError(t, err)
	require.True(t, github.registerCalled())
	require.True(t, secrets.writeCalled())
}

func TestRunHaltsOnConfigReadFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withConfig(config.thatFailsRead()),
	)
	err := cmd.Run(flags.any())

	require.Error(t, err)
	require.False(t, secrets.generateCalled())
}

func TestRunHaltsOnSecretsGenerateFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withSecrets(secrets.thatFailsGenerate()),
	)
	err := cmd.Run(flags.any())

	require.Error(t, err)
	require.False(t, secrets.writeCalled())
}

func TestRunHaltsOnSecretsWriteFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withSecrets(secrets.thatFailsWrite()),
	)
	err := cmd.Run(flags.any())

	require.Error(t, err)
}

func TestRunPropagatesContextResolutionFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withContext(ctx.thatFails()),
	)
	err := cmd.Run(flags.any())

	require.Error(t, err)
	require.False(t, config.writeCalled())
}
