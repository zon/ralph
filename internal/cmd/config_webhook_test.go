package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/webhook"
	"gopkg.in/yaml.v3"
)

// noopFetcher is a collaborators fetcher that always returns an empty list.
func noopFetcher(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

// TestConfigWebhookConfigFlagParsing tests flag parsing for the webhook-config command
func TestConfigWebhookConfigFlagParsing(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedContext   string
		expectedNamespace string
		expectedDryRun    bool
		wantErr           bool
	}{
		{
			name:              "defaults to ralph-webhook namespace",
			args:              []string{"config", "webhook-config"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    false,
		},
		{
			name:              "custom context",
			args:              []string{"config", "webhook-config", "--context", "my-cluster"},
			expectedContext:   "my-cluster",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    false,
		},
		{
			name:              "custom namespace overrides default",
			args:              []string{"config", "webhook-config", "--namespace", "my-ns"},
			expectedContext:   "",
			expectedNamespace: "my-ns",
			expectedDryRun:    false,
		},
		{
			name:              "dry-run flag",
			args:              []string{"config", "webhook-config", "--dry-run"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    true,
		},
		{
			name:              "all flags",
			args:              []string{"config", "webhook-config", "--context", "ctx", "--namespace", "ns", "--dry-run"},
			expectedContext:   "ctx",
			expectedNamespace: "ns",
			expectedDryRun:    true,
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
			assert.Equal(t, tt.expectedDryRun, cmd.Config.WebhookConfig.DryRun)
		})
	}
}

// TestConfigWebhookSecretFlagParsing tests flag parsing for the webhook-secret command
func TestConfigWebhookSecretFlagParsing(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedContext   string
		expectedNamespace string
		expectedDryRun    bool
	}{
		{
			name:              "defaults to ralph-webhook namespace",
			args:              []string{"config", "webhook-secret"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    false,
		},
		{
			name:              "custom context and namespace",
			args:              []string{"config", "webhook-secret", "--context", "prod", "--namespace", "prod-webhook"},
			expectedContext:   "prod",
			expectedNamespace: "prod-webhook",
			expectedDryRun:    false,
		},
		{
			name:              "dry-run flag",
			args:              []string{"config", "webhook-secret", "--dry-run"},
			expectedContext:   "",
			expectedNamespace: "ralph-webhook",
			expectedDryRun:    true,
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
			assert.Equal(t, tt.expectedDryRun, cmd.Config.WebhookSecret.DryRun)
		})
	}
}

// TestBuildWebhookAppConfig tests the pure default-filling logic
func TestBuildWebhookAppConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("fills all defaults when starting from nil", func(t *testing.T) {
		cfg := buildWebhookAppConfig(ctx, nil, "my-repo", "my-owner", "anthropic/claude-sonnet-4-6", noopFetcher)

		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "anthropic/claude-sonnet-4-6", cfg.Model)
		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, "my-owner", cfg.Repos[0].Owner)
		assert.Equal(t, "my-repo", cfg.Repos[0].Name)
	})

	t.Run("does not override existing port", func(t *testing.T) {
		partial := &webhook.AppConfig{Port: 9090}
		cfg := buildWebhookAppConfig(ctx, partial, "repo", "owner", "model", noopFetcher)

		assert.Equal(t, 9090, cfg.Port)
	})

	t.Run("does not override existing model", func(t *testing.T) {
		partial := &webhook.AppConfig{Model: "my-custom-model"}
		cfg := buildWebhookAppConfig(ctx, partial, "repo", "owner", "default-model", noopFetcher)

		assert.Equal(t, "my-custom-model", cfg.Model)
	})

	t.Run("does not duplicate existing repo", func(t *testing.T) {
		partial := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "my-owner", Name: "my-repo"},
			},
		}
		cfg := buildWebhookAppConfig(ctx, partial, "my-repo", "my-owner", "model", noopFetcher)

		require.Len(t, cfg.Repos, 1)
	})

	t.Run("adds detected repo alongside existing repos", func(t *testing.T) {
		partial := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "owner-a", Name: "repo-a"},
			},
		}
		cfg := buildWebhookAppConfig(ctx, partial, "repo-b", "owner-b", "model", noopFetcher)

		require.Len(t, cfg.Repos, 2)
		assert.Equal(t, "repo-a", cfg.Repos[0].Name)
		assert.Equal(t, "repo-b", cfg.Repos[1].Name)
	})

	t.Run("skips repo detection when repoName is empty", func(t *testing.T) {
		cfg := buildWebhookAppConfig(ctx, nil, "", "", "model", noopFetcher)

		assert.Empty(t, cfg.Repos)
	})

	t.Run("loads from partial config file", func(t *testing.T) {
		dir := t.TempDir()
		partialYAML := "port: 7070\n"
		path := filepath.Join(dir, "partial.yaml")
		require.NoError(t, os.WriteFile(path, []byte(partialYAML), 0644))

		loaded, err := webhook.LoadAppConfig(path)
		require.NoError(t, err)

		cfg := buildWebhookAppConfig(ctx, loaded, "my-repo", "my-owner", "default-model", noopFetcher)

		assert.Equal(t, 7070, cfg.Port)
		assert.Equal(t, "default-model", cfg.Model)
	})

	t.Run("populates AllowedUsers from fetcher", func(t *testing.T) {
		fetcher := func(_ context.Context, owner, repo string) ([]string, error) {
			return []string{"alice", "bob"}, nil
		}
		cfg := buildWebhookAppConfig(ctx, nil, "my-repo", "my-owner", "model", fetcher)

		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, []string{"alice", "bob"}, cfg.Repos[0].AllowedUsers)
	})

	t.Run("does not override existing AllowedUsers", func(t *testing.T) {
		partial := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "my-owner", Name: "my-repo", AllowedUsers: []string{"existing-user"}},
			},
		}
		fetcher := func(_ context.Context, owner, repo string) ([]string, error) {
			return []string{"alice", "bob"}, nil
		}
		cfg := buildWebhookAppConfig(ctx, partial, "my-repo", "my-owner", "model", fetcher)

		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, []string{"existing-user"}, cfg.Repos[0].AllowedUsers)
	})

	t.Run("skips AllowedUsers when fetcher returns error", func(t *testing.T) {
		fetcher := func(_ context.Context, owner, repo string) ([]string, error) {
			return nil, fmt.Errorf("API error")
		}
		cfg := buildWebhookAppConfig(ctx, nil, "my-repo", "my-owner", "model", fetcher)

		require.Len(t, cfg.Repos, 1)
		assert.Empty(t, cfg.Repos[0].AllowedUsers)
	})
}

// TestBuildWebhookSecrets tests the pure secret generation logic for the webhook-secret command
func TestBuildWebhookSecrets(t *testing.T) {
	// deterministicGenerator returns predictable secrets for testing
	counter := 0
	deterministicGenerator := func() (string, error) {
		counter++
		return fmt.Sprintf("test-secret-%d", counter), nil
	}

	t.Run("generates a secret for each repo", func(t *testing.T) {
		counter = 0
		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "acme", Name: "repo-a"},
				{Owner: "acme", Name: "repo-b"},
			},
		}

		secrets, err := buildWebhookSecrets(appCfg, deterministicGenerator)
		require.NoError(t, err)

		require.Len(t, secrets.Repos, 2)
		assert.Equal(t, "acme", secrets.Repos[0].Owner)
		assert.Equal(t, "repo-a", secrets.Repos[0].Name)
		assert.Equal(t, "test-secret-1", secrets.Repos[0].WebhookSecret)
		assert.Equal(t, "acme", secrets.Repos[1].Owner)
		assert.Equal(t, "repo-b", secrets.Repos[1].Name)
		assert.Equal(t, "test-secret-2", secrets.Repos[1].WebhookSecret)
	})

	t.Run("returns empty repos list when no repos configured", func(t *testing.T) {
		counter = 0
		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{},
		}

		secrets, err := buildWebhookSecrets(appCfg, deterministicGenerator)
		require.NoError(t, err)

		assert.Empty(t, secrets.Repos)
	})

	t.Run("propagates secret generation errors", func(t *testing.T) {
		failingGenerator := func() (string, error) {
			return "", fmt.Errorf("entropy exhausted")
		}

		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "acme", Name: "repo-a"},
			},
		}

		_, err := buildWebhookSecrets(appCfg, failingGenerator)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entropy exhausted")
	})

	t.Run("generates unique secrets per repo", func(t *testing.T) {
		// Use real generateWebhookSecret to verify uniqueness
		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "acme", Name: "repo-a"},
				{Owner: "acme", Name: "repo-b"},
				{Owner: "acme", Name: "repo-c"},
			},
		}

		secrets, err := buildWebhookSecrets(appCfg, generateWebhookSecret)
		require.NoError(t, err)

		require.Len(t, secrets.Repos, 3)

		// All secrets should be non-empty and unique
		secretSet := make(map[string]bool)
		for _, rs := range secrets.Repos {
			assert.NotEmpty(t, rs.WebhookSecret)
			assert.False(t, secretSet[rs.WebhookSecret], "duplicate secret generated")
			secretSet[rs.WebhookSecret] = true
		}
	})

	t.Run("serializes secrets to valid YAML", func(t *testing.T) {
		counter = 0
		appCfg := &webhook.AppConfig{
			Repos: []webhook.RepoConfig{
				{Owner: "myowner", Name: "myrepo"},
			},
		}

		secrets, err := buildWebhookSecrets(appCfg, deterministicGenerator)
		require.NoError(t, err)

		// Verify that the secrets can be marshaled and unmarshaled as YAML
		secretsBytes, err := yaml.Marshal(secrets)
		require.NoError(t, err)

		var roundTripped webhook.Secrets
		require.NoError(t, yaml.Unmarshal(secretsBytes, &roundTripped))

		require.Len(t, roundTripped.Repos, 1)
		assert.Equal(t, "myowner", roundTripped.Repos[0].Owner)
		assert.Equal(t, "myrepo", roundTripped.Repos[0].Name)
		assert.Equal(t, "test-secret-1", roundTripped.Repos[0].WebhookSecret)
	})
}

// TestGenerateWebhookSecret tests that webhook secrets are cryptographically random
func TestGenerateWebhookSecret(t *testing.T) {
	t.Run("generates a non-empty secret", func(t *testing.T) {
		secret, err := generateWebhookSecret()
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
	})

	t.Run("generates at least 32 characters", func(t *testing.T) {
		secret, err := generateWebhookSecret()
		require.NoError(t, err)
		// 32 bytes base64-encoded is at least 43 characters (raw URL encoding)
		assert.GreaterOrEqual(t, len(secret), 32)
	})

	t.Run("generates unique secrets on repeated calls", func(t *testing.T) {
		secrets := make(map[string]bool)
		for i := 0; i < 10; i++ {
			secret, err := generateWebhookSecret()
			require.NoError(t, err)
			assert.False(t, secrets[secret], "duplicate secret generated on iteration %d", i)
			secrets[secret] = true
		}
	})
}
