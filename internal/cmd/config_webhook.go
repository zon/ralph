package cmd

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/provisioning"
	"github.com/zon/ralph/internal/webhookconfig"
)

type ConfigWebhookConfigCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
	Config    string `help:"Path to a partial AppConfig YAML file to use as a starting point" type:"path" optional:""`
}

type ConfigWebhookSecretCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
}

func (c *ConfigWebhookConfigCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Provisioning webhook-config configmap...")
	fmt.Println()

	kubeContext, err := provisioning.GetKubeContext(ctx, c.Context)
	if err != nil {
		return err
	}

	namespace := c.Namespace

	base := c.readExistingConfigmap(ctx, namespace, kubeContext)

	updates := c.loadConfigUpdates()

	repoName, repoOwner, repoNamespace := c.detectRepoAndNamespace(ctx)

	appCfg := provisioning.BuildWebhookAppConfig(ctx, base, updates, repoOwner, repoName, repoNamespace, provisioning.FetchRepoCollaborators)

	if err := provisioning.WriteWebhookConfigMap(ctx, kubeContext, namespace, appCfg); err != nil {
		return err
	}

	fmt.Printf("ConfigMap '%s' created/updated in namespace '%s'\n", provisioning.WebhookConfigMapName, namespace)
	return nil
}

func (c *ConfigWebhookConfigCmd) readExistingConfigmap(ctx context.Context, namespace, kubeContext string) *webhookconfig.AppConfig {
	existing, err := provisioning.ReadWebhookConfigFromK8s(ctx, namespace, kubeContext)
	if err != nil {
		logger.Warningf("Could not read existing configmap '%s': %v (starting from scratch)", provisioning.WebhookConfigMapName, err)
		return nil
	}
	return existing
}

func (c *ConfigWebhookConfigCmd) loadConfigUpdates() *webhookconfig.AppConfig {
	if c.Config == "" {
		return nil
	}
	loaded, err := webhookconfig.LoadAppConfig(c.Config)
	if err != nil {
		logger.Warningf("Failed to load partial config: %v (ignoring)", err)
		return nil
	}
	return loaded
}

func (c *ConfigWebhookConfigCmd) detectRepoAndNamespace(ctx context.Context) (string, string, string) {
	repo, err := github.GetRepo(ctx)
	if err != nil {
		logger.Warningf("Failed to detect GitHub repository: %v (skipping repo auto-detection)", err)
		return "", "", ""
	}

	if repo.Owner == "" || repo.Name == "" {
		return "", "", ""
	}

	ralphCfg, err := config.LoadConfig()
	if err != nil {
		logger.Warningf("Failed to load .ralph/config.yaml: %v (namespace will be empty)", err)
		return repo.Name, repo.Owner, ""
	}

	return repo.Name, repo.Owner, ralphCfg.Workflow.Namespace
}

func (c *ConfigWebhookSecretCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Provisioning webhook-secrets secret...")
	fmt.Println()

	kubeContext, err := provisioning.GetKubeContext(ctx, c.Context)
	if err != nil {
		return err
	}

	namespace := c.Namespace

	appCfg, err := c.readRepoList(ctx, namespace, kubeContext)
	if err != nil {
		return err
	}

	if err := c.validateRepos(appCfg); err != nil {
		return err
	}

	if err := c.generateAndWriteSecrets(ctx, kubeContext, namespace, appCfg); err != nil {
		return err
	}

	return nil
}

func (c *ConfigWebhookSecretCmd) readRepoList(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error) {
	fmt.Printf("Reading repo list from configmap '%s' in namespace '%s'...\n", provisioning.WebhookConfigMapName, namespace)
	appCfg, err := provisioning.ReadWebhookConfigFromK8s(ctx, namespace, kubeContext)
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook-config: %w\n\nRun 'ralph config webhook-config' first to create the webhook-config configmap.", err)
	}
	return appCfg, nil
}

func (c *ConfigWebhookSecretCmd) validateRepos(appCfg *webhookconfig.AppConfig) error {
	if len(appCfg.Repos) == 0 {
		return fmt.Errorf("no repos found in webhook-config secret — add repos first via 'ralph config webhook-config'")
	}
	fmt.Printf("Found %d repo(s) in webhook-config\n\n", len(appCfg.Repos))
	return nil
}

func (c *ConfigWebhookSecretCmd) generateAndWriteSecrets(ctx context.Context, kubeContext, namespace string, appCfg *webhookconfig.AppConfig) error {
	secrets, err := provisioning.BuildWebhookSecrets(appCfg, provisioning.GenerateWebhookSecret)
	if err != nil {
		return fmt.Errorf("failed to generate webhook secrets: %w", err)
	}

	fmt.Printf("Generated webhook secrets for %d repo(s)\n\n", len(secrets.Repos))

	if err := c.registerWebhooks(ctx, secrets); err != nil {
		return err
	}

	if err := provisioning.WriteWebhookSecrets(ctx, kubeContext, namespace, secrets); err != nil {
		return fmt.Errorf("failed to create/update secret '%s': %w", provisioning.WebhookSecretsSecretName, err)
	}

	logger.Successf("Secret '%s' created/updated in namespace '%s'", provisioning.WebhookSecretsSecretName, namespace)
	return nil
}

func (c *ConfigWebhookSecretCmd) registerWebhooks(ctx context.Context, secrets *webhookconfig.Secrets) error {
	webhookURL := fmt.Sprintf("https://%s/webhook", provisioning.WebhookIngressHostname)
	logger.Infof("Registering webhooks at %s...", webhookURL)
	for _, rs := range secrets.Repos {
		logger.Infof("Registering webhook for %s/%s...", rs.Owner, rs.Name)
		if err := provisioning.RegisterGitHubWebhook(ctx, rs.Owner, rs.Name, webhookURL, rs.WebhookSecret); err != nil {
			logger.Warningf("Failed to register webhook for %s/%s: %v", rs.Owner, rs.Name, err)
		} else {
			logger.Successf("Webhook registered for %s/%s", rs.Owner, rs.Name)
		}
	}
	logger.Info("")
	return nil
}
