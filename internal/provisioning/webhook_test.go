package provisioning

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/webhookconfig"
	"gopkg.in/yaml.v3"
)

func TestBuildWebhookAppConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("fills all defaults from nil base and nil updates", func(t *testing.T) {
		cfg := BuildWebhookAppConfig(ctx, nil, nil, "", "", "", &github.MockGH{})

		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, config.DefaultAppName+"[bot]", cfg.RalphUser)
		assert.Empty(t, cfg.Repos)
	})

	t.Run("preserves base repos when no updates", func(t *testing.T) {
		base := &webhookconfig.AppConfig{
			Port: 9090,
			Repos: []webhookconfig.RepoConfig{
				{Owner: "my-owner", Name: "my-repo", Namespace: "my-ns"},
			},
		}
		cfg := BuildWebhookAppConfig(ctx, base, nil, "", "", "", &github.MockGH{})

		assert.Equal(t, 9090, cfg.Port)
		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, "my-ns", cfg.Repos[0].Namespace)
	})

	t.Run("updates replace matching base repos by owner/name", func(t *testing.T) {
		base := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "repo-a", Namespace: "old-ns", AllowedUsers: []string{"alice"}},
				{Owner: "acme", Name: "repo-b", Namespace: "ns-b"},
			},
		}
		updates := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "repo-a", Namespace: "new-ns"},
			},
		}
		cfg := BuildWebhookAppConfig(ctx, base, updates, "", "", "", &github.MockGH{})

		require.Len(t, cfg.Repos, 2)
		assert.Equal(t, "new-ns", cfg.Repos[0].Namespace)
		assert.Equal(t, "ns-b", cfg.Repos[1].Namespace)
	})

	t.Run("updates add new repos not in base", func(t *testing.T) {
		base := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "repo-a", Namespace: "ns-a"},
			},
		}
		updates := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "repo-b", Namespace: "ns-b"},
			},
		}
		cfg := BuildWebhookAppConfig(ctx, base, updates, "", "", "", &github.MockGH{})

		require.Len(t, cfg.Repos, 2)
		assert.Equal(t, "repo-a", cfg.Repos[0].Name)
		assert.Equal(t, "repo-b", cfg.Repos[1].Name)
	})

	t.Run("updates override port from base", func(t *testing.T) {
		base := &webhookconfig.AppConfig{Port: 8080}
		updates := &webhookconfig.AppConfig{Port: 9090}
		cfg := BuildWebhookAppConfig(ctx, base, updates, "", "", "", &github.MockGH{})

		assert.Equal(t, 9090, cfg.Port)
	})

	t.Run("auto-detected repo upserts into result", func(t *testing.T) {
		cfg := BuildWebhookAppConfig(ctx, nil, nil, "my-owner", "my-repo", "my-ns", &github.MockGH{})

		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, "my-owner", cfg.Repos[0].Owner)
		assert.Equal(t, "my-repo", cfg.Repos[0].Name)
		assert.Equal(t, "my-ns", cfg.Repos[0].Namespace)
	})

	t.Run("auto-detected repo replaces existing entry", func(t *testing.T) {
		base := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "my-owner", Name: "my-repo", Namespace: "old-ns"},
			},
		}
		cfg := BuildWebhookAppConfig(ctx, base, nil, "my-owner", "my-repo", "new-ns", &github.MockGH{})

		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, "new-ns", cfg.Repos[0].Namespace)
	})

	t.Run("auto-detected repo adds alongside existing repos", func(t *testing.T) {
		base := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "owner-a", Name: "repo-a", Namespace: "ns-a"},
			},
		}
		cfg := BuildWebhookAppConfig(ctx, base, nil, "owner-b", "repo-b", "ns-b", &github.MockGH{})

		require.Len(t, cfg.Repos, 2)
		assert.Equal(t, "repo-a", cfg.Repos[0].Name)
		assert.Equal(t, "repo-b", cfg.Repos[1].Name)
		assert.Equal(t, "ns-b", cfg.Repos[1].Namespace)
	})

	t.Run("loads from updates config file", func(t *testing.T) {
		dir := t.TempDir()
		partialYAML := "port: 7070\n"
		path := filepath.Join(dir, "partial.yaml")
		require.NoError(t, os.WriteFile(path, []byte(partialYAML), 0644))

		loaded, err := webhookconfig.LoadAppConfig(path)
		require.NoError(t, err)

		cfg := BuildWebhookAppConfig(ctx, nil, loaded, "", "", "", &github.MockGH{})

		assert.Equal(t, 7070, cfg.Port)
	})

	t.Run("populates AllowedUsers from fetcher", func(t *testing.T) {
		gh := &github.MockGH{
			ListCollaboratorsFn: func(_ context.Context, owner, repo string) ([]string, error) {
				return []string{"alice", "bob"}, nil
			},
		}
		cfg := BuildWebhookAppConfig(ctx, nil, nil, "my-owner", "my-repo", "my-ns", gh)

		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, []string{"alice", "bob"}, cfg.Repos[0].AllowedUsers)
	})

	t.Run("does not override existing AllowedUsers from base", func(t *testing.T) {
		base := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "my-owner", Name: "my-repo", Namespace: "my-ns", AllowedUsers: []string{"existing-user"}},
			},
		}
		gh := &github.MockGH{
			ListCollaboratorsFn: func(_ context.Context, owner, repo string) ([]string, error) {
				return []string{"alice", "bob"}, nil
			},
		}
		cfg := BuildWebhookAppConfig(ctx, base, nil, "", "", "", gh)

		require.Len(t, cfg.Repos, 1)
		assert.Equal(t, []string{"existing-user"}, cfg.Repos[0].AllowedUsers)
	})

	t.Run("skips AllowedUsers when fetcher returns error", func(t *testing.T) {
		gh := &github.MockGH{
			ListCollaboratorsFn: func(_ context.Context, owner, repo string) ([]string, error) {
				return nil, fmt.Errorf("API error")
			},
		}
		cfg := BuildWebhookAppConfig(ctx, nil, nil, "my-owner", "my-repo", "my-ns", gh)

		require.Len(t, cfg.Repos, 1)
		assert.Empty(t, cfg.Repos[0].AllowedUsers)
	})

	t.Run("sets RalphUser to DefaultAppName[bot] by default", func(t *testing.T) {
		cfg := BuildWebhookAppConfig(ctx, nil, nil, "", "", "", &github.MockGH{})

		assert.Equal(t, config.DefaultAppName+"[bot]", cfg.RalphUser)
	})

	t.Run("updates override base RalphUser", func(t *testing.T) {
		base := &webhookconfig.AppConfig{RalphUser: "base-bot"}
		updates := &webhookconfig.AppConfig{RalphUser: "new-bot"}
		cfg := BuildWebhookAppConfig(ctx, base, updates, "", "", "", &github.MockGH{})

		assert.Equal(t, "new-bot", cfg.RalphUser)
	})

	t.Run("base RalphUser preserved when updates has none", func(t *testing.T) {
		base := &webhookconfig.AppConfig{RalphUser: "existing-bot"}
		cfg := BuildWebhookAppConfig(ctx, base, nil, "", "", "", &github.MockGH{})

		assert.Equal(t, "existing-bot", cfg.RalphUser)
	})
}

func TestRegisterGitHubWebhook(t *testing.T) {
	ctx := context.Background()

	t.Run("calls gh.RegisterWebhook with correct arguments", func(t *testing.T) {
		var capturedOwner, capturedRepo, capturedURL, capturedSecret string
		gh := &github.MockGH{
			RegisterWebhookFn: func(_ context.Context, owner, repo, webhookURL, secret string) error {
				capturedOwner = owner
				capturedRepo = repo
				capturedURL = webhookURL
				capturedSecret = secret
				return nil
			},
		}

		err := RegisterGitHubWebhook(ctx, gh, "test-owner", "test-repo", "https://example.com/webhook", "s3kr3t")
		require.NoError(t, err)
		assert.Equal(t, "test-owner", capturedOwner)
		assert.Equal(t, "test-repo", capturedRepo)
		assert.Equal(t, "https://example.com/webhook", capturedURL)
		assert.Equal(t, "s3kr3t", capturedSecret)
	})

	t.Run("propagates error from gh.RegisterWebhook", func(t *testing.T) {
		gh := &github.MockGH{
			RegisterWebhookFn: func(_ context.Context, owner, repo, webhookURL, secret string) error {
				return fmt.Errorf("registration failed")
			},
		}

		err := RegisterGitHubWebhook(ctx, gh, "owner", "repo", "http://hook", "secret")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration failed")
	})
}

func TestBuildWebhookSecrets(t *testing.T) {
	counter := 0
	deterministicGenerator := func() (string, error) {
		counter++
		return fmt.Sprintf("test-secret-%d", counter), nil
	}

	t.Run("generates a secret for each repo", func(t *testing.T) {
		counter = 0
		appCfg := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "repo-a"},
				{Owner: "acme", Name: "repo-b"},
			},
		}

		secrets, err := BuildWebhookSecrets(appCfg, deterministicGenerator)
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
		appCfg := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{},
		}

		secrets, err := BuildWebhookSecrets(appCfg, deterministicGenerator)
		require.NoError(t, err)

		assert.Empty(t, secrets.Repos)
	})

	t.Run("propagates secret generation errors", func(t *testing.T) {
		failingGenerator := func() (string, error) {
			return "", fmt.Errorf("entropy exhausted")
		}

		appCfg := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "repo-a"},
			},
		}

		_, err := BuildWebhookSecrets(appCfg, failingGenerator)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entropy exhausted")
	})

	t.Run("generates unique secrets per repo", func(t *testing.T) {
		appCfg := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "repo-a"},
				{Owner: "acme", Name: "repo-b"},
				{Owner: "acme", Name: "repo-c"},
			},
		}

		secrets, err := BuildWebhookSecrets(appCfg, GenerateWebhookSecret)
		require.NoError(t, err)

		require.Len(t, secrets.Repos, 3)

		secretSet := make(map[string]bool)
		for _, rs := range secrets.Repos {
			assert.NotEmpty(t, rs.WebhookSecret)
			assert.False(t, secretSet[rs.WebhookSecret], "duplicate secret generated")
			secretSet[rs.WebhookSecret] = true
		}
	})

	t.Run("serializes secrets to valid YAML", func(t *testing.T) {
		counter = 0
		appCfg := &webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "myowner", Name: "myrepo"},
			},
		}

		secrets, err := BuildWebhookSecrets(appCfg, deterministicGenerator)
		require.NoError(t, err)

		secretsBytes, err := yaml.Marshal(secrets)
		require.NoError(t, err)

		var roundTripped webhookconfig.Secrets
		require.NoError(t, yaml.Unmarshal(secretsBytes, &roundTripped))

		require.Len(t, roundTripped.Repos, 1)
		assert.Equal(t, "myowner", roundTripped.Repos[0].Owner)
		assert.Equal(t, "myrepo", roundTripped.Repos[0].Name)
		assert.Equal(t, "test-secret-1", roundTripped.Repos[0].WebhookSecret)
	})
}

func TestGetKubeContext(t *testing.T) {
	t.Run("returns contextOverride when non-empty", func(t *testing.T) {
		client := &k8s.MockClient{}
		result, err := GetKubeContext(context.Background(), client, "my-cluster")
		require.NoError(t, err)
		assert.Equal(t, "my-cluster", result)
	})

	t.Run("calls client.GetCurrentContext when contextOverride is empty", func(t *testing.T) {
		client := &k8s.MockClient{
			GetCurrentContextFunc: func(ctx context.Context) (k8s.Context, error) {
				return k8s.Context{Name: "current-cluster"}, nil
			},
		}
		result, err := GetKubeContext(context.Background(), client, "")
		require.NoError(t, err)
		assert.Equal(t, "current-cluster", result)
	})

	t.Run("propagates error from client.GetCurrentContext", func(t *testing.T) {
		client := &k8s.MockClient{
			GetCurrentContextFunc: func(ctx context.Context) (k8s.Context, error) {
				return k8s.Context{}, fmt.Errorf("kubeconfig not found")
			},
		}
		_, err := GetKubeContext(context.Background(), client, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kubeconfig not found")
	})
}

func TestWriteWebhookConfigMap(t *testing.T) {
	t.Run("calls client.CreateOrUpdateConfigMap with serialized config", func(t *testing.T) {
		var capturedName, capturedNamespace, capturedKubeContext string
		var capturedData map[string]string
		client := &k8s.MockClient{
			CreateOrUpdateConfigMapFunc: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
				capturedName = name
				capturedNamespace = namespace
				capturedKubeContext = kubeContext
				capturedData = data
				return nil
			},
		}
		appCfg := webhookconfig.AppConfig{Port: 8080}
		err := WriteWebhookConfigMap(context.Background(), client, "my-ctx", "my-ns", appCfg)
		require.NoError(t, err)
		assert.Equal(t, WebhookConfigMapName, capturedName)
		assert.Equal(t, "my-ns", capturedNamespace)
		assert.Equal(t, "my-ctx", capturedKubeContext)
		assert.Contains(t, capturedData["config.yaml"], "port: 8080")
	})

	t.Run("propagates error from client.CreateOrUpdateConfigMap", func(t *testing.T) {
		client := &k8s.MockClient{
			CreateOrUpdateConfigMapFunc: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
				return fmt.Errorf("API error")
			},
		}
		err := WriteWebhookConfigMap(context.Background(), client, "my-ctx", "my-ns", webhookconfig.AppConfig{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})
}

func TestWriteWebhookSecrets(t *testing.T) {
	t.Run("calls client.CreateOrUpdateSecret with serialized secrets", func(t *testing.T) {
		var capturedName, capturedNamespace, capturedKubeContext string
		var capturedData map[string]string
		client := &k8s.MockClient{
			CreateOrUpdateSecretFunc: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
				capturedName = name
				capturedNamespace = namespace
				capturedKubeContext = kubeContext
				capturedData = data
				return nil
			},
		}
		secrets := &webhookconfig.Secrets{
			Repos: []webhookconfig.RepoSecret{
				{Owner: "acme", Name: "repo-a", WebhookSecret: "s3kr3t"},
			},
		}
		err := WriteWebhookSecrets(context.Background(), client, "my-ctx", "my-ns", secrets)
		require.NoError(t, err)
		assert.Equal(t, WebhookSecretsSecretName, capturedName)
		assert.Equal(t, "my-ns", capturedNamespace)
		assert.Equal(t, "my-ctx", capturedKubeContext)
		assert.Contains(t, capturedData["secrets.yaml"], "s3kr3t")
	})

	t.Run("propagates error from client.CreateOrUpdateSecret", func(t *testing.T) {
		client := &k8s.MockClient{
			CreateOrUpdateSecretFunc: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
				return fmt.Errorf("API error")
			},
		}
		err := WriteWebhookSecrets(context.Background(), client, "my-ctx", "my-ns", &webhookconfig.Secrets{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})
}

func TestGenerateWebhookSecret(t *testing.T) {
	t.Run("generates a non-empty secret", func(t *testing.T) {
		secret, err := GenerateWebhookSecret()
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
	})

	t.Run("generates at least 32 characters", func(t *testing.T) {
		secret, err := GenerateWebhookSecret()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(secret), 32)
	})

	t.Run("generates unique secrets on repeated calls", func(t *testing.T) {
		secrets := make(map[string]bool)
		for i := 0; i < 10; i++ {
			secret, err := GenerateWebhookSecret()
			require.NoError(t, err)
			assert.False(t, secrets[secret], "duplicate secret generated on iteration %d", i)
			secrets[secret] = true
		}
	})
}
