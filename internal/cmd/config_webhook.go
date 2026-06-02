package cmd

import (
	"context"
	"os"

	"github.com/zon/ralph/internal/config"
	internalgithub "github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/orchestration/config/webhook"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/provisioning"
	"github.com/zon/ralph/internal/webhookconfig"
)

type ConfigWebhookConfigCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
	Config    string `help:"Path to a partial AppConfig YAML file to use as a starting point" type:"path" optional:""`
	out       *output.Client
}

type ConfigWebhookSecretCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
	out       *output.Client
}

func (c *ConfigWebhookConfigCmd) Run() error {
	ctx := context.Background()

	if c.out == nil {
		c.out = output.NewClient(os.Stdout, os.Stderr, false)
	}

	orchestrator := newConfigWebhookOrchestrator(c.out)
	return orchestrator.RunConfig(ctx, c.Config, c.Context, c.Namespace)
}

func (c *ConfigWebhookSecretCmd) Run() error {
	ctx := context.Background()

	if c.out == nil {
		c.out = output.NewClient(os.Stdout, os.Stderr, false)
	}

	orchestrator := newConfigWebhookOrchestrator(c.out)
	return orchestrator.RunSecret(ctx, c.Context, c.Namespace)
}

type configWebhookConfigLoaderAdapter struct{}

func (a *configWebhookConfigLoaderAdapter) Load() (*config.RalphConfig, error) {
	return config.LoadConfig()
}

type configWebhookGitHubClientAdapter struct {
	out *output.Client
}

func (a *configWebhookGitHubClientAdapter) GetRepo(ctx context.Context) (string, string, error) {
	repo, err := internalgithub.GetRepo(ctx)
	if err != nil {
		return "", "", err
	}
	return repo.Owner, repo.Name, nil
}

func (a *configWebhookGitHubClientAdapter) RegisterWebhook(ctx context.Context, owner, repo, webhookURL, secret string) error {
	gh := internalgithub.NewGH(a.out)
	return gh.RegisterWebhook(ctx, owner, repo, webhookURL, secret)
}

type configWebhookK8sClientAdapter struct{}

func (a *configWebhookK8sClientAdapter) GetCurrentContext(ctx context.Context) (webhook.K8sContext, error) {
	realClient := k8s.NewClient()
	k8sCtx, err := realClient.GetCurrentContext(ctx)
	if err != nil {
		return webhook.K8sContext{}, err
	}
	return webhook.K8sContext{Name: k8sCtx.Name, Namespace: k8sCtx.Namespace}, nil
}

func (a *configWebhookK8sClientAdapter) CreateOrUpdateConfigMap(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	return k8s.NewClient().CreateOrUpdateConfigMap(ctx, name, namespace, kubeContext, data)
}

func (a *configWebhookK8sClientAdapter) CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	return k8s.NewClient().CreateOrUpdateSecret(ctx, name, namespace, kubeContext, data)
}

type configWebhookConfigMapReaderAdapter struct{}

func (a *configWebhookConfigMapReaderAdapter) ReadWebhookConfig(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error) {
	return provisioning.ReadWebhookConfigFromK8s(ctx, namespace, kubeContext)
}

type configWebhookAppConfigLoaderAdapter struct{}

func (a *configWebhookAppConfigLoaderAdapter) LoadAppConfig(path string) (*webhookconfig.AppConfig, error) {
	return webhookconfig.LoadAppConfig(path)
}

type configWebhookAppConfigBuilderAdapter struct {
	out *output.Client
}

func (a *configWebhookAppConfigBuilderAdapter) Build(ctx context.Context, base, updates *webhookconfig.AppConfig, repoOwner, repoName, repoNamespace string) webhookconfig.AppConfig {
	gh := internalgithub.NewGH(a.out)
	return provisioning.BuildWebhookAppConfig(ctx, a.out, base, updates, repoOwner, repoName, repoNamespace, gh)
}

type configWebhookSecretBuilderAdapter struct{}

func (a *configWebhookSecretBuilderAdapter) BuildSecrets(appCfg *webhookconfig.AppConfig) (*webhookconfig.Secrets, error) {
	return provisioning.BuildWebhookSecrets(appCfg, provisioning.GenerateWebhookSecret)
}

type configWebhookLoggerAdapter struct {
	out *output.Client
}

func (a *configWebhookLoggerAdapter) Info(msg string) {
	a.out.Info(msg)
}

func (a *configWebhookLoggerAdapter) Infof(format string, args ...interface{}) {
	a.out.Infof(format, args...)
}

func (a *configWebhookLoggerAdapter) Warnf(format string, args ...interface{}) {
	a.out.Warnf(format, args...)
}

func (a *configWebhookLoggerAdapter) Success(msg string) {
	a.out.Success(msg)
}

func (a *configWebhookLoggerAdapter) Successf(format string, args ...interface{}) {
	a.out.Successf(format, args...)
}

func (a *configWebhookLoggerAdapter) Debugf(format string, args ...interface{}) {
	a.out.Debugf(format, args...)
}

func newConfigWebhookOrchestrator(out *output.Client) *webhook.Cmd {
	return webhook.New(
		&configWebhookConfigLoaderAdapter{},
		&configWebhookGitHubClientAdapter{out: out},
		&configWebhookK8sClientAdapter{},
		&configWebhookConfigMapReaderAdapter{},
		&configWebhookAppConfigLoaderAdapter{},
		&configWebhookAppConfigBuilderAdapter{out: out},
		&configWebhookSecretBuilderAdapter{},
		&configWebhookLoggerAdapter{out: out},
	)
}
