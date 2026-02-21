package webhook

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RepoConfig represents a single repository entry in the app config
type RepoConfig struct {
	Owner     string `yaml:"owner"`
	Name      string `yaml:"name"`
	ClonePath string `yaml:"clonePath"`
}

// AppConfig is the application configuration loaded from a YAML file
type AppConfig struct {
	Port          int          `yaml:"port"`
	RalphUsername string       `yaml:"ralphUsername"`
	Repos         []RepoConfig `yaml:"repos"`
	Model         string       `yaml:"model"`
}

// RepoSecret holds the webhook secret for a single repository
type RepoSecret struct {
	Owner         string `yaml:"owner"`
	Name          string `yaml:"name"`
	WebhookSecret string `yaml:"webhookSecret"`
}

// Secrets holds all secrets loaded from the secrets YAML file
type Secrets struct {
	Repos []RepoSecret `yaml:"repos"`
}

// Config is the fully-loaded service configuration combining AppConfig and Secrets
type Config struct {
	App     AppConfig
	Secrets Secrets
}

// LoadAppConfig loads the application configuration from a YAML file
func LoadAppConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read app config file %s: %w", path, err)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse app config YAML: %w", err)
	}

	return &cfg, nil
}

// LoadSecrets loads the secrets from a YAML file
func LoadSecrets(path string) (*Secrets, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets file %s: %w", path, err)
	}

	var s Secrets
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse secrets YAML: %w", err)
	}

	return &s, nil
}

// LoadConfig loads and validates the full service configuration.
// configPath and secretsPath may be empty; in that case the values from
// the environment variables WEBHOOK_CONFIG and WEBHOOK_SECRETS are used.
func LoadConfig(configPath, secretsPath string) (*Config, error) {
	if configPath == "" {
		configPath = os.Getenv("WEBHOOK_CONFIG")
	}
	if secretsPath == "" {
		secretsPath = os.Getenv("WEBHOOK_SECRETS")
	}

	if configPath == "" {
		return nil, fmt.Errorf("app config path is required (set --config flag or WEBHOOK_CONFIG env var)")
	}
	if secretsPath == "" {
		return nil, fmt.Errorf("secrets path is required (set --secrets flag or WEBHOOK_SECRETS env var)")
	}

	appCfg, err := LoadAppConfig(configPath)
	if err != nil {
		return nil, err
	}

	secrets, err := LoadSecrets(secretsPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		App:     *appCfg,
		Secrets: *secrets,
	}

	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ValidateConfig validates that required secrets are present
func ValidateConfig(cfg *Config) error {
	// Build a lookup of repo secrets for validation
	secretsByRepo := make(map[string]string)
	for _, rs := range cfg.Secrets.Repos {
		key := repoKey(rs.Owner, rs.Name)
		secretsByRepo[key] = rs.WebhookSecret
	}

	// Every repo in config must have a webhook secret
	for _, repo := range cfg.App.Repos {
		key := repoKey(repo.Owner, repo.Name)
		if secretsByRepo[key] == "" {
			return fmt.Errorf("required secret missing: no webhook secret configured for repo %s/%s", repo.Owner, repo.Name)
		}
	}

	return nil
}

// WebhookSecretForRepo returns the webhook secret for the given owner/name pair.
// Returns an empty string if no secret is found.
func (c *Config) WebhookSecretForRepo(owner, name string) string {
	key := repoKey(owner, name)
	for _, rs := range c.Secrets.Repos {
		if repoKey(rs.Owner, rs.Name) == key {
			return rs.WebhookSecret
		}
	}
	return ""
}

// RepoByFullName looks up a RepoConfig by owner and name.
// Returns nil if not found.
func (c *Config) RepoByFullName(owner, name string) *RepoConfig {
	key := repoKey(owner, name)
	for i := range c.App.Repos {
		if repoKey(c.App.Repos[i].Owner, c.App.Repos[i].Name) == key {
			return &c.App.Repos[i]
		}
	}
	return nil
}

func repoKey(owner, name string) string {
	return owner + "/" + name
}
