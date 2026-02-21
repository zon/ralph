package webhook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

const validAppConfig = `
port: 8080
ralphUsername: ralph-bot
model: anthropic/claude-sonnet-4-6
repos:
  - owner: acme
    name: my-service
    clonePath: /repos/my-service
  - owner: acme
    name: another-service
    clonePath: /repos/another-service
`

const validSecrets = `
githubToken: ghp_testtoken123
repos:
  - owner: acme
    name: my-service
    webhookSecret: secret-abc
  - owner: acme
    name: another-service
    webhookSecret: secret-xyz
`

func TestLoadAppConfig(t *testing.T) {
	t.Run("loads valid config", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "config.yaml", validAppConfig)

		cfg, err := LoadAppConfig(path)
		require.NoError(t, err)

		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "ralph-bot", cfg.RalphUsername)
		assert.Equal(t, "anthropic/claude-sonnet-4-6", cfg.Model)
		require.Len(t, cfg.Repos, 2)
		assert.Equal(t, "acme", cfg.Repos[0].Owner)
		assert.Equal(t, "my-service", cfg.Repos[0].Name)
		assert.Equal(t, "/repos/my-service", cfg.Repos[0].ClonePath)
	})

	t.Run("error on missing file", func(t *testing.T) {
		_, err := LoadAppConfig("/nonexistent/config.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read app config file")
	})

	t.Run("error on invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "config.yaml", "port: [invalid yaml\n")

		_, err := LoadAppConfig(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse app config YAML")
	})
}

func TestLoadSecrets(t *testing.T) {
	t.Run("loads valid secrets", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "secrets.yaml", validSecrets)

		s, err := LoadSecrets(path)
		require.NoError(t, err)

		assert.Equal(t, "ghp_testtoken123", s.GitHubToken)
		require.Len(t, s.Repos, 2)
		assert.Equal(t, "acme", s.Repos[0].Owner)
		assert.Equal(t, "my-service", s.Repos[0].Name)
		assert.Equal(t, "secret-abc", s.Repos[0].WebhookSecret)
	})

	t.Run("error on missing file", func(t *testing.T) {
		_, err := LoadSecrets("/nonexistent/secrets.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read secrets file")
	})

	t.Run("error on invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "secrets.yaml", "githubToken: [bad\n")

		_, err := LoadSecrets(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse secrets YAML")
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("loads from explicit paths", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := writeFile(t, dir, "config.yaml", validAppConfig)
		secPath := writeFile(t, dir, "secrets.yaml", validSecrets)

		cfg, err := LoadConfig(cfgPath, secPath)
		require.NoError(t, err)

		assert.Equal(t, 8080, cfg.App.Port)
		assert.Equal(t, "ralph-bot", cfg.App.RalphUsername)
		assert.Equal(t, "ghp_testtoken123", cfg.Secrets.GitHubToken)
		require.Len(t, cfg.App.Repos, 2)
	})

	t.Run("loads from environment variables", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := writeFile(t, dir, "config.yaml", validAppConfig)
		secPath := writeFile(t, dir, "secrets.yaml", validSecrets)

		t.Setenv("WEBHOOK_CONFIG", cfgPath)
		t.Setenv("WEBHOOK_SECRETS", secPath)

		cfg, err := LoadConfig("", "")
		require.NoError(t, err)

		assert.Equal(t, 8080, cfg.App.Port)
		assert.Equal(t, "ghp_testtoken123", cfg.Secrets.GitHubToken)
	})

	t.Run("CLI flags take precedence over env vars", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := writeFile(t, dir, "config.yaml", validAppConfig)
		secPath := writeFile(t, dir, "secrets.yaml", validSecrets)

		// Set env vars pointing at non-existent paths
		t.Setenv("WEBHOOK_CONFIG", "/nonexistent/env-config.yaml")
		t.Setenv("WEBHOOK_SECRETS", "/nonexistent/env-secrets.yaml")

		// Explicit paths should win
		cfg, err := LoadConfig(cfgPath, secPath)
		require.NoError(t, err)
		assert.Equal(t, 8080, cfg.App.Port)
	})

	t.Run("error when config path is missing and no env var", func(t *testing.T) {
		t.Setenv("WEBHOOK_CONFIG", "")
		t.Setenv("WEBHOOK_SECRETS", "")

		_, err := LoadConfig("", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "app config path is required")
	})

	t.Run("error when secrets path is missing and no env var", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := writeFile(t, dir, "config.yaml", validAppConfig)

		t.Setenv("WEBHOOK_CONFIG", "")
		t.Setenv("WEBHOOK_SECRETS", "")

		_, err := LoadConfig(cfgPath, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secrets path is required")
	})

	t.Run("error when secrets file missing", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := writeFile(t, dir, "config.yaml", validAppConfig)

		_, err := LoadConfig(cfgPath, "/nonexistent/secrets.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read secrets file")
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("valid config passes", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Port:          8080,
				RalphUsername: "ralph-bot",
				Repos: []RepoConfig{
					{Owner: "acme", Name: "my-service", ClonePath: "/repos/my-service"},
				},
			},
			Secrets: Secrets{
				GitHubToken: "ghp_testtoken",
				Repos: []RepoSecret{
					{Owner: "acme", Name: "my-service", WebhookSecret: "webhook-secret-1"},
				},
			},
		}

		err := ValidateConfig(cfg)
		require.NoError(t, err)
	})

	t.Run("error when githubToken is missing", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Repos: []RepoConfig{
					{Owner: "acme", Name: "my-service"},
				},
			},
			Secrets: Secrets{
				GitHubToken: "",
				Repos: []RepoSecret{
					{Owner: "acme", Name: "my-service", WebhookSecret: "secret"},
				},
			},
		}

		err := ValidateConfig(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "githubToken")
	})

	t.Run("error when repo has no webhook secret", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Repos: []RepoConfig{
					{Owner: "acme", Name: "my-service"},
					{Owner: "acme", Name: "other-service"},
				},
			},
			Secrets: Secrets{
				GitHubToken: "ghp_token",
				Repos: []RepoSecret{
					{Owner: "acme", Name: "my-service", WebhookSecret: "secret"},
					// other-service has no secret entry
				},
			},
		}

		err := ValidateConfig(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "acme/other-service")
	})

	t.Run("no repos is valid", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Port: 8080,
			},
			Secrets: Secrets{
				GitHubToken: "ghp_token",
			},
		}

		err := ValidateConfig(cfg)
		require.NoError(t, err)
	})
}

func TestWebhookSecretForRepo(t *testing.T) {
	cfg := &Config{
		Secrets: Secrets{
			Repos: []RepoSecret{
				{Owner: "acme", Name: "my-service", WebhookSecret: "secret-abc"},
				{Owner: "acme", Name: "other", WebhookSecret: "secret-xyz"},
			},
		},
	}

	t.Run("returns secret for known repo", func(t *testing.T) {
		secret := cfg.WebhookSecretForRepo("acme", "my-service")
		assert.Equal(t, "secret-abc", secret)
	})

	t.Run("returns empty string for unknown repo", func(t *testing.T) {
		secret := cfg.WebhookSecretForRepo("acme", "nonexistent")
		assert.Equal(t, "", secret)
	})
}

func TestRepoByFullName(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Repos: []RepoConfig{
				{Owner: "acme", Name: "my-service", ClonePath: "/repos/my-service"},
				{Owner: "acme", Name: "other", ClonePath: "/repos/other"},
			},
		},
	}

	t.Run("returns repo for known owner/name", func(t *testing.T) {
		repo := cfg.RepoByFullName("acme", "my-service")
		require.NotNil(t, repo)
		assert.Equal(t, "/repos/my-service", repo.ClonePath)
	})

	t.Run("returns nil for unknown repo", func(t *testing.T) {
		repo := cfg.RepoByFullName("acme", "nonexistent")
		assert.Nil(t, repo)
	})
}
