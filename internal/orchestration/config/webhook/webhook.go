package webhook

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/webhookconfig"
	"gopkg.in/yaml.v3"
)

const (
	webhookConfigMapName     = "webhook-config"
	webhookSecretsSecretName = "webhook-secrets"
	webhookIngressHostname   = "ralph.haralovich.org"
)

type ConfigLoader interface {
	Load() (*config.RalphConfig, error)
}

type GitHubClient interface {
	GetRepo(ctx context.Context) (owner, name string, err error)
	RegisterWebhook(ctx context.Context, owner, repo, webhookURL, secret string) error
}

type K8sContext struct {
	Name      string
	Namespace string
}

type K8sClient interface {
	GetCurrentContext(ctx context.Context) (K8sContext, error)
	CreateOrUpdateConfigMap(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
	CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
}

type ConfigMapReader interface {
	ReadWebhookConfig(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error)
}

type AppConfigLoader interface {
	LoadAppConfig(path string) (*webhookconfig.AppConfig, error)
}

type AppConfigBuilder interface {
	Build(ctx context.Context, base, updates *webhookconfig.AppConfig, repoOwner, repoName, repoNamespace string) webhookconfig.AppConfig
}

type SecretBuilder interface {
	BuildSecrets(appCfg *webhookconfig.AppConfig) (*webhookconfig.Secrets, error)
}

type Logger interface {
	Info(msg string)
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Success(msg string)
	Successf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type Cmd struct {
	configLoader     ConfigLoader
	ghClient         GitHubClient
	k8sClient        K8sClient
	configMapReader  ConfigMapReader
	appConfigLoader  AppConfigLoader
	appConfigBuilder AppConfigBuilder
	secretBuilder    SecretBuilder
	log              Logger
}

func New(
	configLoader ConfigLoader,
	ghClient GitHubClient,
	k8sClient K8sClient,
	configMapReader ConfigMapReader,
	appConfigLoader AppConfigLoader,
	appConfigBuilder AppConfigBuilder,
	secretBuilder SecretBuilder,
	log Logger,
) *Cmd {
	return &Cmd{
		configLoader:     configLoader,
		ghClient:         ghClient,
		k8sClient:        k8sClient,
		configMapReader:  configMapReader,
		appConfigLoader:  appConfigLoader,
		appConfigBuilder: appConfigBuilder,
		secretBuilder:    secretBuilder,
		log:              log,
	}
}

func (c *Cmd) RunConfig(ctx context.Context, configPath, flagContext, flagNamespace string) error {
	c.log.Info("Provisioning webhook-config configmap...")

	cfg, err := c.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := c.resolveKubeContext(ctx, cfg, flagContext, flagNamespace)
	if err != nil {
		return err
	}

	namespace := k8sCtx.Namespace
	base := c.readExistingConfigmap(ctx, namespace, k8sCtx.Name)
	updates := c.loadConfigUpdates(configPath)
	repoName, repoOwner, repoNamespace := c.detectRepoAndNamespace(ctx)

	appCfg := c.appConfigBuilder.Build(ctx, base, updates, repoOwner, repoName, repoNamespace)

	cfgBytes, err := yaml.Marshal(appCfg)
	if err != nil {
		return fmt.Errorf("failed to serialize AppConfig to YAML: %w", err)
	}

	configMapData := map[string]string{
		"config.yaml": string(cfgBytes),
	}

	if err := c.k8sClient.CreateOrUpdateConfigMap(ctx, webhookConfigMapName, namespace, k8sCtx.Name, configMapData); err != nil {
		return fmt.Errorf("failed to create/update configmap '%s': %w", webhookConfigMapName, err)
	}

	c.log.Infof("ConfigMap '%s' created/updated in namespace '%s'", webhookConfigMapName, namespace)
	return nil
}

func (c *Cmd) RunSecret(ctx context.Context, flagContext, flagNamespace string) error {
	c.log.Info("Provisioning webhook-secrets secret...")

	cfg, err := c.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := c.resolveKubeContext(ctx, cfg, flagContext, flagNamespace)
	if err != nil {
		return err
	}

	namespace := k8sCtx.Namespace

	appCfg, err := c.readRepoList(ctx, namespace, k8sCtx.Name)
	if err != nil {
		return err
	}

	if err := c.validateRepos(appCfg); err != nil {
		return err
	}

	if err := c.generateAndWriteSecrets(ctx, k8sCtx.Name, namespace, appCfg); err != nil {
		return err
	}

	return nil
}

func (c *Cmd) resolveKubeContext(ctx context.Context, ralphConfig *config.RalphConfig, flagContext, flagNamespace string) (K8sContext, error) {
	var k8sCtx K8sContext

	if flagContext != "" {
		c.log.Debugf("Using Kubernetes context: %s", flagContext)
		k8sCtx.Name = flagContext
	} else if ralphConfig != nil && ralphConfig.Workflow.Context != "" {
		c.log.Debugf("Using context from .ralph/config.yaml: %s", ralphConfig.Workflow.Context)
		k8sCtx.Name = ralphConfig.Workflow.Context
	} else {
		current, err := c.k8sClient.GetCurrentContext(ctx)
		if err != nil {
			return K8sContext{}, fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured.", err)
		}
		c.log.Debugf("Using current Kubernetes context: %s", current.Name)
		k8sCtx.Name = current.Name
		k8sCtx.Namespace = current.Namespace
	}

	if flagNamespace != "" {
		c.log.Debugf("Using namespace: %s", flagNamespace)
		k8sCtx.Namespace = flagNamespace
	} else if ralphConfig != nil && ralphConfig.Workflow.Namespace != "" {
		c.log.Debugf("Using namespace from .ralph/config.yaml: %s", ralphConfig.Workflow.Namespace)
		k8sCtx.Namespace = ralphConfig.Workflow.Namespace
	} else if ralphConfig != nil && ralphConfig.ConfigPath != "" {
		c.log.Debugf("Using default namespace: %s (config found)", "config")
		k8sCtx.Namespace = "config"
	}

	if k8sCtx.Namespace == "" {
		c.log.Debugf("Using namespace: %s (default)", "default")
		k8sCtx.Namespace = "default"
	}

	return k8sCtx, nil
}

func (c *Cmd) readExistingConfigmap(ctx context.Context, namespace, kubeContext string) *webhookconfig.AppConfig {
	existing, err := c.configMapReader.ReadWebhookConfig(ctx, namespace, kubeContext)
	if err != nil {
		c.log.Warnf("Could not read existing configmap '%s': %v (starting from scratch)", webhookConfigMapName, err)
		return nil
	}
	return existing
}

func (c *Cmd) loadConfigUpdates(path string) *webhookconfig.AppConfig {
	if path == "" {
		return nil
	}
	loaded, err := c.appConfigLoader.LoadAppConfig(path)
	if err != nil {
		c.log.Warnf("Failed to load partial config: %v (ignoring)", err)
		return nil
	}
	return loaded
}

func (c *Cmd) detectRepoAndNamespace(ctx context.Context) (string, string, string) {
	owner, name, err := c.ghClient.GetRepo(ctx)
	if err != nil {
		c.log.Warnf("Failed to detect GitHub repository: %v (skipping repo auto-detection)", err)
		return "", "", ""
	}

	if owner == "" || name == "" {
		return "", "", ""
	}

	ralphCfg, err := c.configLoader.Load()
	if err != nil {
		c.log.Warnf("Failed to load .ralph/config.yaml: %v (namespace will be empty)", err)
		return name, owner, ""
	}

	return name, owner, ralphCfg.Workflow.Namespace
}

func (c *Cmd) readRepoList(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error) {
	c.log.Infof("Reading repo list from configmap '%s' in namespace '%s'...", webhookConfigMapName, namespace)
	appCfg, err := c.configMapReader.ReadWebhookConfig(ctx, namespace, kubeContext)
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook-config: %w\n\nRun 'ralph config webhook-config' first to create the webhook-config configmap.", err)
	}
	return appCfg, nil
}

func (c *Cmd) validateRepos(appCfg *webhookconfig.AppConfig) error {
	if len(appCfg.Repos) == 0 {
		return fmt.Errorf("no repos found in webhook-config secret — add repos first via 'ralph config webhook-config'")
	}
	c.log.Infof("Found %d repo(s) in webhook-config", len(appCfg.Repos))
	return nil
}

func (c *Cmd) generateAndWriteSecrets(ctx context.Context, kubeContext, namespace string, appCfg *webhookconfig.AppConfig) error {
	secrets, err := c.secretBuilder.BuildSecrets(appCfg)
	if err != nil {
		return fmt.Errorf("failed to generate webhook secrets: %w", err)
	}

	c.log.Infof("Generated webhook secrets for %d repo(s)", len(secrets.Repos))

	c.registerWebhooks(ctx, secrets)

	secretsBytes, err := yaml.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("failed to serialize Secrets to YAML: %w", err)
	}

	secretData := map[string]string{
		"secrets.yaml": string(secretsBytes),
	}

	if err := c.k8sClient.CreateOrUpdateSecret(ctx, webhookSecretsSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret '%s': %w", webhookSecretsSecretName, err)
	}

	c.log.Successf("Secret '%s' created/updated in namespace '%s'", webhookSecretsSecretName, namespace)
	return nil
}

func (c *Cmd) registerWebhooks(ctx context.Context, secrets *webhookconfig.Secrets) {
	webhookURL := fmt.Sprintf("https://%s/webhook", webhookIngressHostname)
	c.log.Infof("Registering webhooks at %s...", webhookURL)
	for _, rs := range secrets.Repos {
		c.log.Infof("Registering webhook for %s/%s...", rs.Owner, rs.Name)
		if err := c.ghClient.RegisterWebhook(ctx, rs.Owner, rs.Name, webhookURL, rs.WebhookSecret); err != nil {
			c.log.Warnf("Failed to register webhook for %s/%s: %v", rs.Owner, rs.Name, err)
		} else {
			c.log.Successf("Webhook registered for %s/%s", rs.Owner, rs.Name)
		}
	}
}
