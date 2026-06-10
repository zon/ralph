package webhookconfig

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/output"
	"gopkg.in/yaml.v3"
)

const (
	WebhookConfigMapName     = "webhook-config"
	WebhookSecretsSecretName = "webhook-secrets"
	WebhookIngressHostname   = "ralph.haralovich.org"
)

func mergeRepo(repos []RepoConfig, incoming RepoConfig) []RepoConfig {
	for i, r := range repos {
		if r.Owner == incoming.Owner && r.Name == incoming.Name {
			repos[i] = incoming
			return repos
		}
	}
	return append(repos, incoming)
}

func BuildWebhookAppConfig(ctx context.Context, out *output.Client, base, updates *AppConfig, repoOwner, repoName, repoNamespace string, gh github.GHClient) AppConfig {
	var cfg AppConfig

	if base != nil {
		cfg = *base
	}

	if updates != nil {
		if updates.Port != 0 {
			cfg.Port = updates.Port
		}
		if updates.RalphUser != "" {
			cfg.RalphUser = updates.RalphUser
		}
		if updates.CommentInstructionsFile != "" {
			cfg.CommentInstructionsFile = updates.CommentInstructionsFile
		}
		for _, r := range updates.Repos {
			cfg.Repos = mergeRepo(cfg.Repos, r)
		}
	}

	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	if cfg.RalphUser == "" {
		cfg.RalphUser = config.DefaultAppName + "[bot]"
	}

	if repoOwner != "" && repoName != "" {
		cfg.Repos = mergeRepo(cfg.Repos, RepoConfig{
			Owner:     repoOwner,
			Name:      repoName,
			Namespace: repoNamespace,
		})
	}

	for i, r := range cfg.Repos {
		if len(r.AllowedUsers) == 0 {
			users, err := gh.ListCollaborators(ctx, r.Owner, r.Name)
			if err != nil {
				if out != nil {
					out.Warnf("Failed to fetch collaborators for %s/%s: %v (skipping AllowedUsers)", r.Owner, r.Name, err)
				}
			} else {
				cfg.Repos[i].AllowedUsers = users
			}
		}
	}

	return cfg
}

func BuildWebhookAppConfigFromK8s(ctx context.Context, namespace, kubeContext, configPath string, client k8s.Client, ghClient github.GHClient, out *output.Client) AppConfig {
	base, err := ReadWebhookConfigFromK8s(ctx, client, namespace, kubeContext)
	if err != nil {
		if out != nil {
			out.Warnf("Could not read existing configmap '%s': %v (starting from scratch)", WebhookConfigMapName, err)
		}
		base = nil
	}

	var updates *AppConfig
	if configPath != "" {
		loaded, err := LoadAppConfig(configPath)
		if err != nil {
			if out != nil {
				out.Warnf("Failed to load partial config: %v (ignoring)", err)
			}
		} else {
			updates = loaded
		}
	}

	return BuildWebhookAppConfig(ctx, out, base, updates, "", "", "", ghClient)
}

func GenerateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func ReadWebhookConfigFromK8s(ctx context.Context, client k8s.Client, namespace, kubeContext string) (*AppConfig, error) {
	raw, err := client.GetConfigMapData(ctx, WebhookConfigMapName, namespace, kubeContext)
	if err != nil {
		return nil, fmt.Errorf("failed to read configmap '%s' from namespace '%s': %w", WebhookConfigMapName, namespace, err)
	}

	var appCfg AppConfig
	if err := yaml.Unmarshal([]byte(raw), &appCfg); err != nil {
		return nil, fmt.Errorf("failed to parse AppConfig YAML from configmap: %w", err)
	}

	return &appCfg, nil
}

func RegisterGitHubWebhook(ctx context.Context, gh github.GHClient, owner, repo, webhookURL, secret string) error {
	return gh.RegisterWebhook(ctx, owner, repo, webhookURL, secret)
}

func RegisterAllGitHubWebhooks(ctx context.Context, ghClient github.GHClient, out *output.Client, repos []RepoSecret) {
	webhookURL := fmt.Sprintf("https://%s/webhook", WebhookIngressHostname)
	out.Infof("Registering webhooks at %s...", webhookURL)
	for _, rs := range repos {
		if err := RegisterGitHubWebhook(ctx, ghClient, rs.Owner, rs.Name, webhookURL, rs.WebhookSecret); err != nil {
			out.Warnf("Failed to register webhook for %s/%s: %v", rs.Owner, rs.Name, err)
		} else {
			out.Successf("Webhook registered for %s/%s", rs.Owner, rs.Name)
		}
	}
	out.Info("")
}

func BuildWebhookSecrets(appCfg *AppConfig, secretGenerator func() (string, error)) (*Secrets, error) {
	secrets := &Secrets{}

	for _, repo := range appCfg.Repos {
		secret, err := secretGenerator()
		if err != nil {
			return nil, fmt.Errorf("failed to generate secret for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		secrets.Repos = append(secrets.Repos, RepoSecret{
			Owner:         repo.Owner,
			Name:          repo.Name,
			WebhookSecret: secret,
		})
	}

	return secrets, nil
}

func WriteWebhookConfigMap(ctx context.Context, client k8s.Client, kubeContext, namespace string, appCfg AppConfig) error {
	cfgBytes, err := yaml.Marshal(appCfg)
	if err != nil {
		return fmt.Errorf("failed to serialize AppConfig to YAML: %w", err)
	}

	cfgYAML := string(cfgBytes)

	configMapData := map[string]string{
		"config.yaml": cfgYAML,
	}

	if err := client.CreateOrUpdateConfigMap(ctx, WebhookConfigMapName, namespace, kubeContext, configMapData); err != nil {
		return fmt.Errorf("failed to create/update configmap '%s': %w", WebhookConfigMapName, err)
	}

	return nil
}

func WriteWebhookSecretsAndLog(ctx context.Context, client k8s.Client, kubeContext, namespace string, secrets *Secrets, out *output.Client) error {
	if err := WriteWebhookSecrets(ctx, client, kubeContext, namespace, secrets); err != nil {
		return err
	}
	out.Successf("Secret '%s' created/updated in namespace '%s'", WebhookSecretsSecretName, namespace)
	return nil
}

func WriteWebhookSecrets(ctx context.Context, client k8s.Client, kubeContext, namespace string, secrets *Secrets) error {
	secretsBytes, err := yaml.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("failed to serialize Secrets to YAML: %w", err)
	}

	secretsYAML := string(secretsBytes)

	secretData := map[string]string{
		"secrets.yaml": secretsYAML,
	}

	if err := client.CreateOrUpdateSecret(ctx, WebhookSecretsSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret '%s': %w", WebhookSecretsSecretName, err)
	}

	return nil
}

func GetKubeContext(ctx context.Context, client k8s.Client, contextOverride string) (string, error) {
	if contextOverride != "" {
		return contextOverride, nil
	}
	currentCtx, err := client.GetCurrentContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current Kubernetes context: %w", err)
	}
	return currentCtx.Name, nil
}
