package provisioning

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/webhookconfig"
	"gopkg.in/yaml.v3"
)

const (
	WebhookConfigMapName     = "webhook-config"
	WebhookSecretsSecretName = "webhook-secrets"
	WebhookIngressHostname   = "ralph.haralovich.org"
)

func FetchRepoCollaborators(ctx context.Context, owner, repo string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/%s/collaborators", owner, repo),
		"--jq", ".[].login",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list collaborators for %s/%s: %w (stderr: %s)",
			owner, repo, err, stderr.String())
	}

	var logins []string
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			logins = append(logins, line)
		}
	}
	return logins, nil
}

func mergeRepo(repos []webhookconfig.RepoConfig, incoming webhookconfig.RepoConfig) []webhookconfig.RepoConfig {
	for i, r := range repos {
		if r.Owner == incoming.Owner && r.Name == incoming.Name {
			repos[i] = incoming
			return repos
		}
	}
	return append(repos, incoming)
}

func BuildWebhookAppConfig(ctx context.Context, base, updates *webhookconfig.AppConfig, repoOwner, repoName, repoNamespace string, fetcher func(context.Context, string, string) ([]string, error)) webhookconfig.AppConfig {
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
			users, err := fetcher(ctx, r.Owner, r.Name)
			if err != nil {
				logger.Warningf("Failed to fetch collaborators for %s/%s: %v (skipping AllowedUsers)", r.Owner, r.Name, err)
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

func RegisterGitHubWebhook(ctx context.Context, owner, repo, webhookURL, secret string) error {
	listArgs := []string{
		"api",
		fmt.Sprintf("repos/%s/%s/hooks", owner, repo),
		"--jq", fmt.Sprintf(`.[] | select(.config.url | contains("%s")) | .id`, webhookURL),
	}
	listCmdFull := exec.CommandContext(ctx, "gh", listArgs...)
	var listOut, listErr bytes.Buffer
	listCmdFull.Stdout = &listOut
	listCmdFull.Stderr = &listErr

	var existingID string
	if err := listCmdFull.Run(); err == nil {
		existingID = strings.TrimSpace(listOut.String())
	}

	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"push", "pull_request", "pull_request_review", "issue_comment"},
		"config": map[string]string{
			"url":          webhookURL,
			"content_type": "json",
			"secret":       secret,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	if existingID != "" && existingID != "null" {
		updateCmd := exec.CommandContext(ctx, "gh", "api",
			fmt.Sprintf("repos/%s/%s/hooks/%s", owner, repo, existingID),
			"--method", "PATCH",
			"--input", "-",
		)
		updateCmd.Stdin = bytes.NewReader(payloadBytes)
		var updateErr bytes.Buffer
		updateCmd.Stderr = &updateErr
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to update webhook for %s/%s: %w (stderr: %s)",
				owner, repo, err, updateErr.String())
		}
		return nil
	}

	createCmd := exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/%s/hooks", owner, repo),
		"--method", "POST",
		"--input", "-",
	)
	createCmd.Stdin = bytes.NewReader(payloadBytes)
	var createErr bytes.Buffer
	createCmd.Stderr = &createErr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create webhook for %s/%s: %w (stderr: %s)",
			owner, repo, err, createErr.String())
	}

	return nil
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

func WriteWebhookConfigMap(ctx context.Context, kubeContext, namespace string, appCfg webhookconfig.AppConfig) error {
	cfgBytes, err := yaml.Marshal(appCfg)
	if err != nil {
		return fmt.Errorf("failed to serialize AppConfig to YAML: %w", err)
	}

	cfgYAML := string(cfgBytes)

	configMapData := map[string]string{
		"config.yaml": cfgYAML,
	}

	if err := k8s.CreateOrUpdateConfigMap(ctx, WebhookConfigMapName, namespace, kubeContext, configMapData); err != nil {
		return fmt.Errorf("failed to create/update configmap '%s': %w", WebhookConfigMapName, err)
	}

	return nil
}

func WriteWebhookSecrets(ctx context.Context, kubeContext, namespace string, secrets *webhookconfig.Secrets) error {
	secretsBytes, err := yaml.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("failed to serialize Secrets to YAML: %w", err)
	}

	secretsYAML := string(secretsBytes)

	secretData := map[string]string{
		"secrets.yaml": secretsYAML,
	}

	if err := k8s.CreateOrUpdateSecret(ctx, WebhookSecretsSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret '%s': %w", WebhookSecretsSecretName, err)
	}

	return nil
}

func GetKubeContext(ctx context.Context, contextOverride string) (string, error) {
	if contextOverride != "" {
		return contextOverride, nil
	}
	currentCtx, err := k8s.GetCurrentContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current Kubernetes context: %w", err)
	}
	return currentCtx.Name, nil
}
