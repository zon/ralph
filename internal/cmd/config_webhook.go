package cmd

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
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/webhook"
	"gopkg.in/yaml.v3"
)

// webhookConfigMapName is the name of the Kubernetes ConfigMap for the webhook app config
const webhookConfigMapName = "webhook-config"

// webhookSecretsSecretName is the name of the Kubernetes secret for the webhook secrets
const webhookSecretsSecretName = "webhook-secrets"

// webhookIngressHostname is the default ingress hostname used to match webhooks on GitHub
const webhookIngressHostname = "ralph.haralovich.org"

// ConfigWebhookConfigCmd provisions the webhook-config Kubernetes secret
type ConfigWebhookConfigCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
	Config    string `help:"Path to a partial AppConfig YAML file to use as a starting point" type:"path" optional:""`
	DryRun    bool   `help:"Simulate execution without making changes" default:"false"`
}

// ConfigWebhookSecretCmd provisions the webhook-secrets Kubernetes secret
type ConfigWebhookSecretCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
	DryRun    bool   `help:"Simulate execution without making changes" default:"false"`
}

// WebhookConfigResult holds the result of building an AppConfig for dry-run inspection
type WebhookConfigResult struct {
	AppConfig webhook.AppConfig
	YAML      string
}

// WebhookSecretsResult holds the result of building Secrets for dry-run inspection
type WebhookSecretsResult struct {
	Secrets webhook.Secrets
	YAML    string
}

// collaboratorsFetcher is a function that fetches repo collaborator logins.
// It is a variable so tests can substitute a fake implementation.
var collaboratorsFetcher = fetchRepoCollaborators

// fetchRepoCollaborators uses the gh CLI to list all collaborators for the given repo.
// It returns the list of login names, or an error.
func fetchRepoCollaborators(ctx context.Context, owner, repo string) ([]string, error) {
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

// mergeRepo upserts a RepoConfig into the repos slice, replacing any existing entry
// with the same owner/name. Returns the updated slice.
func mergeRepo(repos []webhook.RepoConfig, incoming webhook.RepoConfig) []webhook.RepoConfig {
	for i, r := range repos {
		if r.Owner == incoming.Owner && r.Name == incoming.Name {
			repos[i] = incoming
			return repos
		}
	}
	return append(repos, incoming)
}

// buildWebhookAppConfig builds an AppConfig by layering updates onto a base config.
// base is the existing config (e.g. read from the k8s configmap); may be nil.
// updates is an optional set of changes to apply on top (e.g. from --config flag); may be nil.
// Repos in updates replace matching repos in base (by owner/name); new repos are appended.
// repoOwner, repoName, and repoNamespace are the auto-detected repo details; if non-empty,
// this repo is upserted last (so it can be updated without a --config file).
// fetcher is the function used to look up repo collaborators (injectable for tests).
func buildWebhookAppConfig(ctx context.Context, base, updates *webhook.AppConfig, repoOwner, repoName, repoNamespace string, fetcher func(context.Context, string, string) ([]string, error)) webhook.AppConfig {
	var cfg webhook.AppConfig

	// Start from base (existing configmap contents)
	if base != nil {
		cfg = *base
	}

	// Apply updates on top: scalar fields and repo upserts
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

	// Fill in port default
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	// Set the ralph bot user if not already configured
	if cfg.RalphUser == "" {
		cfg.RalphUser = config.DefaultAppName + "[bot]"
	}

	// Upsert auto-detected repo
	if repoOwner != "" && repoName != "" {
		cfg.Repos = mergeRepo(cfg.Repos, webhook.RepoConfig{
			Owner:     repoOwner,
			Name:      repoName,
			Namespace: repoNamespace,
		})
	}

	// Populate AllowedUsers for repos that don't already have them configured.
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

// Run executes the config webhook command
func (c *ConfigWebhookConfigCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Provisioning webhook-config configmap...")
	fmt.Println()

	// Determine Kubernetes context (namespace defaults to ralph-webhook via struct tag)
	var kubeContext string
	if c.Context != "" {
		kubeContext = c.Context
	} else {
		currentCtx, err := k8s.GetCurrentContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current Kubernetes context: %w", err)
		}
		kubeContext = currentCtx
	}

	namespace := c.Namespace

	// Read existing configmap as the base (ignore error if it doesn't exist yet)
	var base *webhook.AppConfig
	if existing, err := readWebhookConfigFromK8s(ctx, namespace, kubeContext); err != nil {
		logger.Warningf("Could not read existing configmap '%s': %v (starting from scratch)", webhookConfigMapName, err)
	} else {
		base = existing
	}

	// Load optional updates from --config flag
	var updates *webhook.AppConfig
	if c.Config != "" {
		loaded, err := webhook.LoadAppConfig(c.Config)
		if err != nil {
			return fmt.Errorf("failed to load partial config: %w", err)
		}
		updates = loaded
	}

	// Auto-detect repo and namespace from the working directory
	repoName, repoOwner, err := github.GetRepo(ctx)
	if err != nil {
		logger.Warningf("Failed to detect GitHub repository: %v (skipping repo auto-detection)", err)
		repoName = ""
		repoOwner = ""
	}

	var repoNamespace string
	if repoOwner != "" && repoName != "" {
		ralphCfg, err := config.LoadConfig()
		if err != nil {
			logger.Warningf("Failed to load .ralph/config.yaml: %v (namespace will be empty)", err)
		} else {
			repoNamespace = ralphCfg.Workflow.Namespace
		}
	}

	// Build AppConfig by merging base → updates → auto-detected repo
	appCfg := buildWebhookAppConfig(ctx, base, updates, repoOwner, repoName, repoNamespace, collaboratorsFetcher)

	// Serialize to YAML
	cfgBytes, err := yaml.Marshal(appCfg)
	if err != nil {
		return fmt.Errorf("failed to serialize AppConfig to YAML: %w", err)
	}

	cfgYAML := string(cfgBytes)
	fmt.Printf("AppConfig YAML:\n%s\n", cfgYAML)

	if c.DryRun {
		fmt.Printf("Dry run: would write configmap '%s' in namespace '%s' (context: %s)\n", webhookConfigMapName, namespace, kubeContext)
		return nil
	}

	// Write to Kubernetes ConfigMap
	configMapData := map[string]string{
		"config.yaml": cfgYAML,
	}

	if err := k8s.CreateOrUpdateConfigMap(ctx, webhookConfigMapName, namespace, kubeContext, configMapData); err != nil {
		return fmt.Errorf("failed to create/update configmap '%s': %w", webhookConfigMapName, err)
	}

	fmt.Printf("ConfigMap '%s' created/updated in namespace '%s'\n", webhookConfigMapName, namespace)
	return nil
}

// generateWebhookSecret generates a cryptographically random webhook secret (32 bytes, base64url-encoded)
func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// readWebhookConfigFromK8s reads the webhook-config ConfigMap from Kubernetes and returns
// the AppConfig it contains.
func readWebhookConfigFromK8s(ctx context.Context, namespace, kubeContext string) (*webhook.AppConfig, error) {
	// kubectl get configmap webhook-config -n <ns> --context <ctx> -o jsonpath='{.data.config\.yaml}'
	args := []string{
		"get", "configmap", webhookConfigMapName,
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
			webhookConfigMapName, namespace, err, stderr.String())
	}

	// ConfigMap data values are stored as plain text (not base64-encoded like Secrets)
	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		return nil, fmt.Errorf("configmap '%s' exists but config.yaml key is empty", webhookConfigMapName)
	}

	var appCfg webhook.AppConfig
	if err := yaml.Unmarshal([]byte(raw), &appCfg); err != nil {
		return nil, fmt.Errorf("failed to parse AppConfig YAML from configmap: %w", err)
	}

	return &appCfg, nil
}

// registerGitHubWebhook registers (or updates) a webhook on GitHub for the given repo
// using the provided secret. It looks for an existing webhook matching the ingress hostname
// and updates it; otherwise it creates a new one.
func registerGitHubWebhook(ctx context.Context, owner, repo, webhookURL, secret string) error {
	// List hooks and find one matching our URL
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

	// Build the webhook payload
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
		// Update existing webhook
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

	// Create new webhook
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

// buildWebhookSecrets generates cryptographically random webhook secrets for each repo
// in the provided AppConfig. It uses the provided secretGenerator function to allow
// testing with predictable values.
func buildWebhookSecrets(appCfg *webhook.AppConfig, secretGenerator func() (string, error)) (*webhook.Secrets, error) {
	secrets := &webhook.Secrets{}

	for _, repo := range appCfg.Repos {
		secret, err := secretGenerator()
		if err != nil {
			return nil, fmt.Errorf("failed to generate secret for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		secrets.Repos = append(secrets.Repos, webhook.RepoSecret{
			Owner:         repo.Owner,
			Name:          repo.Name,
			WebhookSecret: secret,
		})
	}

	return secrets, nil
}

// Run executes the config webhook-secret command
func (c *ConfigWebhookSecretCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Provisioning webhook-secrets secret...")
	fmt.Println()

	// Determine Kubernetes context (namespace defaults to ralph-webhook via struct tag)
	var kubeContext string
	if c.Context != "" {
		kubeContext = c.Context
	} else {
		currentCtx, err := k8s.GetCurrentContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current Kubernetes context: %w", err)
		}
		kubeContext = currentCtx
	}

	namespace := c.Namespace

	// Read repo list from webhook-config ConfigMap
	fmt.Printf("Reading repo list from configmap '%s' in namespace '%s'...\n", webhookConfigMapName, namespace)
	appCfg, err := readWebhookConfigFromK8s(ctx, namespace, kubeContext)
	if err != nil {
		return fmt.Errorf("failed to read webhook-config: %w\n\nRun 'ralph config webhook-config' first to create the webhook-config configmap.", err)
	}

	if len(appCfg.Repos) == 0 {
		return fmt.Errorf("no repos found in webhook-config secret — add repos first via 'ralph config webhook-config'")
	}

	fmt.Printf("Found %d repo(s) in webhook-config\n\n", len(appCfg.Repos))

	// Generate webhook secrets for each repo
	secrets, err := buildWebhookSecrets(appCfg, generateWebhookSecret)
	if err != nil {
		return fmt.Errorf("failed to generate webhook secrets: %w", err)
	}

	// Serialize to YAML
	secretsBytes, err := yaml.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("failed to serialize Secrets to YAML: %w", err)
	}

	secretsYAML := string(secretsBytes)
	fmt.Printf("Generated webhook secrets for %d repo(s)\n\n", len(secrets.Repos))

	if c.DryRun {
		fmt.Printf("Dry run: would write secret '%s' in namespace '%s' (context: %s)\n", webhookSecretsSecretName, namespace, kubeContext)
		for _, rs := range secrets.Repos {
			fmt.Printf("  - %s/%s: (secret generated, not shown in dry-run)\n", rs.Owner, rs.Name)
		}
		return nil
	}

	// Register webhooks on GitHub for each repo
	webhookURL := fmt.Sprintf("https://%s/webhook", webhookIngressHostname)
	fmt.Printf("Registering webhooks at %s...\n", webhookURL)
	for _, rs := range secrets.Repos {
		fmt.Printf("  Registering webhook for %s/%s...\n", rs.Owner, rs.Name)
		if err := registerGitHubWebhook(ctx, rs.Owner, rs.Name, webhookURL, rs.WebhookSecret); err != nil {
			logger.Warningf("Failed to register webhook for %s/%s: %v", rs.Owner, rs.Name, err)
			fmt.Printf("  ⚠ Failed to register webhook for %s/%s: %v\n", rs.Owner, rs.Name, err)
		} else {
			fmt.Printf("  ✓ Webhook registered for %s/%s\n", rs.Owner, rs.Name)
		}
	}
	fmt.Println()

	// Write to Kubernetes secret
	secretData := map[string]string{
		"secrets.yaml": secretsYAML,
	}

	if err := k8s.CreateOrUpdateSecret(ctx, webhookSecretsSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret '%s': %w", webhookSecretsSecretName, err)
	}

	fmt.Printf("Secret '%s' created/updated in namespace '%s'\n", webhookSecretsSecretName, namespace)
	return nil
}
