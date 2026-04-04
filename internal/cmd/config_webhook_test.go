package cmd

import (
	"context"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/webhookconfig"
)

func TestConfigWebhookConfigFlagParsing(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedContext   string
		expectedNamespace string
		wantErr           bool
	}{
		{
			name:              "defaults to ralph-webhook namespace",
			args:              []string{"config", "webhook"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
		},
		{
			name:              "custom context",
			args:              []string{"config", "webhook", "--context", "my-cluster"},
			expectedContext:   "my-cluster",
			expectedNamespace: "ralph-webhook",
		},
		{
			name:              "custom namespace overrides default",
			args:              []string{"config", "webhook", "--namespace", "my-ns"},
			expectedContext:   "",
			expectedNamespace: "my-ns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.expectedContext, cmd.Config.WebhookConfig.Context)
			assert.Equal(t, tt.expectedNamespace, cmd.Config.WebhookConfig.Namespace)
		})
	}
}

func TestConfigWebhookSecretFlagParsing(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedContext   string
		expectedNamespace string
	}{
		{
			name:              "defaults to ralph-webhook namespace",
			args:              []string{"config", "webhook-secret"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
		},
		{
			name:              "custom context and namespace",
			args:              []string{"config", "webhook-secret", "--context", "prod", "--namespace", "prod-webhook"},
			expectedContext:   "prod",
			expectedNamespace: "prod-webhook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Cmd{}
			parser, err := kong.New(cmd,
				kong.Name("ralph"),
				kong.Exit(func(int) {}),
			)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedContext, cmd.Config.WebhookSecret.Context)
			assert.Equal(t, tt.expectedNamespace, cmd.Config.WebhookSecret.Namespace)
		})
	}
}

func TestConfigWebhookSecretCmd_RegisterWebhooks_NoRepos(t *testing.T) {
	cmd := &ConfigWebhookSecretCmd{}

	err := cmd.registerWebhooks(context.Background(), &webhookconfig.Secrets{
		Repos: []webhookconfig.RepoSecret{},
	})

	require.NoError(t, err)
}

func TestConfigWebhookSecretCmd_RegisterWebhooks_MultipleRepos(t *testing.T) {
	cmd := &ConfigWebhookSecretCmd{}

	err := cmd.registerWebhooks(context.Background(), &webhookconfig.Secrets{
		Repos: []webhookconfig.RepoSecret{
			{Owner: "owner1", Name: "repo1", WebhookSecret: "secret1"},
			{Owner: "owner2", Name: "repo2", WebhookSecret: "secret2"},
			{Owner: "owner3", Name: "repo3", WebhookSecret: "secret3"},
		},
	})

	require.NoError(t, err)
}
