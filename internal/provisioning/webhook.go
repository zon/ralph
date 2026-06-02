package provisioning

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/webhookconfig"
	"gopkg.in/yaml.v3"
)

const (
	WebhookConfigMapName     = "webhook-config"
	WebhookSecretsSecretName = "webhook-secrets"
	WebhookIngressHostname   = "ralph.haralovich.org"
)

func mergeRepo(repos []webhookconfig.RepoConfig, incoming webhookconfig.RepoConfig) []webhookconfig.RepoConfig {
	for i, r := range repos {
		if r.Owner == incoming.Owner && r.Name == incoming.Name {
			repos[i] = incoming
			return repos
		}
	}
	return append(repos, incoming)
}

func BuildWebhookAppConfig(ctx context.Context, out *output.Client, base, updates *webhookconfig.AppConfig, repoOwner, repoName, repoNamespace string, gh github.GHClient) webhookconfig.AppConfig {
	var cfg webhookconfig.AppConfig

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
		cfg.Repos = mergeRepo(cfg.Repos, webhookconfig.RepoConfig{
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

func GenerateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func ReadWebhookConfigFromK8s(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error) {
	args := []string{
		"get", "configmap", WebhookConfigMapName,
		"-n", namespace,
		"-o", `jsonpath={.data.config\.yaml}`,
	}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to read configmap '%s' from namespace '%s': %w (stderr: %s)",
			WebhookConfigMapName, namespace, err, stderr.String())
	}

	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		return nil, fmt.Errorf("configmap '%s' exists but config.yaml key is empty", WebhookConfigMapName)
	}

	var appCfg webhookconfig.AppConfig
	if err := yaml.Unmarshal([]byte(raw), &appCfg); err != nil {
		return nil, fmt.Errorf("failed to parse AppConfig YAML from configmap: %w", err)
	}

	return &appCfg, nil
}

func RegisterGitHubWebhook(ctx context.Context, gh github.GHClient, owner, repo, webhookURL, secret string) error {
	return gh.RegisterWebhook(ctx, owner, repo, webhookURL, secret)
}

func BuildWebhookSecrets(appCfg *webhookconfig.AppConfig, secretGenerator func() (string, error)) (*webhookconfig.Secrets, error) {
	secrets := &webhookconfig.Secrets{}

	for _, repo := range appCfg.Repos {
		secret, err := secretGenerator()
		if err != nil {
			return nil, fmt.Errorf("failed to generate secret for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		secrets.Repos = append(secrets.Repos, webhookconfig.RepoSecret{
			Owner:         repo.Owner,
			Name:          repo.Name,
			WebhookSecret: secret,
		})
	}

	return secrets, nil
}

func WriteWebhookConfigMap(ctx context.Context, client k8s.Client, kubeContext, namespace string, appCfg webhookconfig.AppConfig) error {
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

func WriteWebhookSecrets(ctx context.Context, client k8s.Client, kubeContext, namespace string, secrets *webhookconfig.Secrets) error {
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
