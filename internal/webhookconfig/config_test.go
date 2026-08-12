package webhookconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

const validAppConfig = `
port: 8080
repos:
  - owner: acme
    name: my-service
    namespace: ns-a
  - owner: acme
    name: another-service
    namespace: ns-b
`

const validSecrets = `
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
		require.Len(t, cfg.Repos, 2)
		assert.Equal(t, "acme", cfg.Repos[0].Owner)
		assert.Equal(t, "my-service", cfg.Repos[0].Name)
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

	t.Run("defaults to embedded comment instructions", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "config.yaml", validAppConfig)

		cfg, err := LoadAppConfig(path)
		require.NoError(t, err)
		assert.Equal(t, config.DefaultCommentInstructions(), cfg.CommentInstructions)
	})

	t.Run("loads comment instructions from file", func(t *testing.T) {
		dir := t.TempDir()
		instrPath := writeFile(t, dir, "comment.md", "# Custom comment instructions")
		yaml := "port: 8080\ncommentInstructionsFile: " + instrPath + "\n"
		path := writeFile(t, dir, "config.yaml", yaml)

		cfg, err := LoadAppConfig(path)
		require.NoError(t, err)
		assert.Equal(t, "# Custom comment instructions", cfg.CommentInstructions)
	})

	t.Run("error on missing comment instructions file", func(t *testing.T) {
		dir := t.TempDir()
		yaml := "port: 8080\ncommentInstructionsFile: /nonexistent/comment.md\n"
		path := writeFile(t, dir, "config.yaml", yaml)

		_, err := LoadAppConfig(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read commentInstructionsFile")
	})
}

// Scenario: Comment instructions override.
//
// GIVEN `commentInstructionsFile` points to a custom markdown file
// WHEN a comment event is dispatched
// THEN the custom file's content is used as the AI prompt template instead of
// the built-in default.
func TestLoadAppConfig_CommentInstructionsOverride(t *testing.T) {
	dir := t.TempDir()
	customInstructions := "# Custom comment instructions\n\nReply in a terse style."
	instrPath := writeFile(t, dir, "comment.md", customInstructions)
	yaml := "port: 8080\ncommentInstructionsFile: " + instrPath + "\n"
	path := writeFile(t, dir, "config.yaml", yaml)

	cfg, err := LoadAppConfig(path)
	require.NoError(t, err)
	assert.Equal(t, customInstructions, cfg.CommentInstructions, "the custom file's content must be used as the AI prompt template")
	assert.NotEqual(t, config.DefaultCommentInstructions(), cfg.CommentInstructions, "the built-in default must not be used")
}

// Item test: the webhook service config no longer accepts a merge instructions
// file setting, and loading a config that still names one does not fail.
func TestLoadConfig_MergeInstructionsFileSettingIgnored(t *testing.T) {
	dir := t.TempDir()
	cfgYAML := "port: 8080\nmergeInstructionsFile: /nonexistent/merge-instructions.md\nrepos:\n  - owner: acme\n    name: my-service\n    namespace: ns-a\n"
	cfgPath := writeFile(t, dir, "config.yaml", cfgYAML)
	secPath := writeFile(t, dir, "secrets.yaml", validSecrets)

	cfg, err := LoadConfig(cfgPath, secPath)
	require.NoError(t, err, "loading a config that still names a merge instructions file must not fail")
	assert.Equal(t, config.DefaultCommentInstructions(), cfg.App.CommentInstructions, "the merge instructions file must not be loaded")
}

// Item test: comment instructions loading and overriding is unchanged — a
// config that still names a merge instructions file keeps its comment override.
func TestLoadAppConfig_CommentOverrideUnchangedWhenMergeFileNamed(t *testing.T) {
	dir := t.TempDir()
	customInstructions := "# Custom comment instructions"
	instrPath := writeFile(t, dir, "comment.md", customInstructions)
	yaml := "port: 8080\ncommentInstructionsFile: " + instrPath + "\nmergeInstructionsFile: /nonexistent/merge-instructions.md\n"
	path := writeFile(t, dir, "config.yaml", yaml)

	cfg, err := LoadAppConfig(path)
	require.NoError(t, err)
	assert.Equal(t, customInstructions, cfg.CommentInstructions, "comment instructions must still override from the custom file")
}

func TestLoadSecrets(t *testing.T) {
	t.Run("loads valid secrets", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "secrets.yaml", validSecrets)

		s, err := LoadSecrets(path)
		require.NoError(t, err)

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
	tests := []struct {
		name        string
		cfg         *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config passes",
			cfg: &Config{
				App: AppConfig{
					Port: 8080,
					Repos: []RepoConfig{
						{Owner: "acme", Name: "my-service", Namespace: "team-ns"},
					},
				},
				Secrets: Secrets{
					Repos: []RepoSecret{
						{Owner: "acme", Name: "my-service", WebhookSecret: "webhook-secret-1"},
					},
				},
			},
		},
		{
			name: "error when repo has no namespace",
			cfg: &Config{
				App: AppConfig{
					Repos: []RepoConfig{
						{Owner: "acme", Name: "my-service"},
					},
				},
				Secrets: Secrets{
					Repos: []RepoSecret{
						{Owner: "acme", Name: "my-service", WebhookSecret: "secret"},
					},
				},
			},
			wantErr:     true,
			errContains: "namespace is required",
		},
		{
			name: "error when repo has no webhook secret",
			cfg: &Config{
				App: AppConfig{
					Repos: []RepoConfig{
						{Owner: "acme", Name: "my-service", Namespace: "ns-a"},
						{Owner: "acme", Name: "other-service", Namespace: "ns-b"},
					},
				},
				Secrets: Secrets{
					Repos: []RepoSecret{
						{Owner: "acme", Name: "my-service", WebhookSecret: "secret"},
					},
				},
			},
			wantErr:     true,
			errContains: "acme/other-service",
		},
		{
			name: "no repos is valid",
			cfg: &Config{
				App: AppConfig{
					Port: 8080,
				},
				Secrets: Secrets{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConfig(tc.cfg)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
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

	tests := []struct {
		name     string
		owner    string
		repoName string
		want     string
	}{
		{name: "returns secret for known repo", owner: "acme", repoName: "my-service", want: "secret-abc"},
		{name: "returns empty string for unknown repo", owner: "acme", repoName: "nonexistent", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			secret := cfg.WebhookSecretForRepo(tc.owner, tc.repoName)
			assert.Equal(t, tc.want, secret)
		})
	}
}

func TestIsUserAllowed(t *testing.T) {
	tests := []struct {
		name         string
		allowedUsers []string
		username     string
		want         bool
	}{
		{"empty list allows all", nil, "anyone", true},
		{"empty list allows all (empty slice)", []string{}, "anyone", true},
		{"user in list is allowed", []string{"alice", "bob"}, "alice", true},
		{"user not in list is denied", []string{"alice", "bob"}, "charlie", false},
		{"comparison is case-insensitive", []string{"Alice"}, "alice", true},
		{"comparison is case-insensitive (upper)", []string{"alice"}, "ALICE", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &RepoConfig{AllowedUsers: tc.allowedUsers}
			got := repo.IsUserAllowed(tc.username)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsUserIgnored(t *testing.T) {
	tests := []struct {
		name         string
		ralphUser    string
		ignoredUsers []string
		username     string
		want         bool
	}{
		{"no ralph user and empty list ignores no one", "", nil, "anyone", false},
		{"ralph user is always ignored", "zralphen", nil, "zralphen", true},
		{"ralph user is ignored regardless of per-repo list", "zralphen", []string{"other-bot"}, "zralphen", true},
		{"ralph user comparison is case-insensitive", "Zralphen", nil, "zralphen", true},
		{"ralph user comparison is case-insensitive (upper)", "zralphen", nil, "ZRALPHEN", true},
		{"user in per-repo list is ignored", "", []string{"zralphen", "bot"}, "zralphen", true},
		{"user not in ralph user or list is not ignored", "zralphen", nil, "alice", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{App: AppConfig{RalphUser: tc.ralphUser}}
			repo := &RepoConfig{IgnoredUsers: tc.ignoredUsers}
			got := cfg.IsUserIgnored(repo, tc.username)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRepoConfig_Namespace(t *testing.T) {
	const configWithNamespace = `
port: 8080
repos:
  - owner: acme
    name: my-service
    namespace: team-ns
  - owner: acme
    name: another-service
`
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yaml", configWithNamespace)

	cfg, err := LoadAppConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "team-ns", cfg.Repos[0].Namespace)
	assert.Equal(t, "", cfg.Repos[1].Namespace)
}

func TestRepoByFullName(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Repos: []RepoConfig{
				{Owner: "acme", Name: "my-service"},
				{Owner: "acme", Name: "other"},
			},
		},
	}

	tests := []struct {
		name     string
		owner    string
		repoName string
		wantNil  bool
	}{
		{name: "returns repo for known owner/name", owner: "acme", repoName: "my-service", wantNil: false},
		{name: "returns nil for unknown repo", owner: "acme", repoName: "nonexistent", wantNil: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := cfg.RepoByFullName(tc.owner, tc.repoName)
			if tc.wantNil {
				assert.Nil(t, repo)
			} else {
				require.NotNil(t, repo)
			}
		})
	}
}
