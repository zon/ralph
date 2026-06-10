package main

import (
	"context"
	"os"

	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	webhooksetconfig "github.com/zon/ralph/internal/orchestration/webhooksetconfig"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/webhookconfig"
)

type SetCmd struct {
	Config SetConfigCmd `cmd:"" help:"Set webhook configuration and secrets"`
}

type SetConfigCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
	Config    string `name:"partial-config" help:"Path to a partial AppConfig YAML file to use as a starting point" type:"path" optional:""`
}

func (c *SetConfigCmd) Run() error {
	ctx := context.Background()
	out := output.NewClient(os.Stdout, os.Stderr, false)

	k8sClient := k8s.NewClient()
	ghClient := github.NewGH(out)

	cmd := &webhooksetconfig.SetConfigCmd{
		Ctx:     &setconfigCtxClient{ctx: ctx, k8sClient: k8sClient},
		Config:  &setconfigCfgClient{ctx: ctx, k8sClient: k8sClient, ghClient: ghClient, out: out},
		Secrets: &setconfigSecretsClient{ctx: ctx, k8sClient: k8sClient, out: out},
		GitHub:  &setconfigGitHubClient{ctx: ctx, ghClient: ghClient, out: out},
	}

	return cmd.Run(webhooksetconfig.Flags{
		Context:    c.Context,
		Namespace:  c.Namespace,
		ConfigPath: c.Config,
	})
}

type setconfigCtxClient struct {
	ctx       context.Context
	k8sClient k8s.Client
}

func (c *setconfigCtxClient) Resolve(flagContext, flagNamespace string) (webhooksetconfig.K8sContext, error) {
	kubeCtx, err := webhookconfig.GetKubeContext(c.ctx, c.k8sClient, flagContext)
	if err != nil {
		return webhooksetconfig.K8sContext{}, err
	}
	return webhooksetconfig.K8sContext{Name: kubeCtx, Namespace: flagNamespace}, nil
}

type setconfigCfgClient struct {
	ctx       context.Context
	k8sClient k8s.Client
	ghClient  github.GHClient
	out       *output.Client
}

func (c *setconfigCfgClient) Build(k8sCtx webhooksetconfig.K8sContext, configPath string) webhookconfig.AppConfig {
	return webhookconfig.BuildWebhookAppConfigFromK8s(c.ctx, k8sCtx.Namespace, k8sCtx.Name, configPath, c.k8sClient, c.ghClient, c.out)
}

func (c *setconfigCfgClient) Write(k8sCtx webhooksetconfig.K8sContext, cfg webhookconfig.AppConfig) error {
	return webhookconfig.WriteWebhookConfigMap(c.ctx, c.k8sClient, k8sCtx.Name, k8sCtx.Namespace, cfg)
}

func (c *setconfigCfgClient) Read(k8sCtx webhooksetconfig.K8sContext) (webhookconfig.AppConfig, error) {
	cfg, err := webhookconfig.ReadWebhookConfigFromK8s(c.ctx, c.k8sClient, k8sCtx.Namespace, k8sCtx.Name)
	if err != nil {
		return webhookconfig.AppConfig{}, err
	}
	return *cfg, nil
}

type setconfigSecretsClient struct {
	ctx       context.Context
	k8sClient k8s.Client
	out       *output.Client
}

func (c *setconfigSecretsClient) Generate(cfg webhookconfig.AppConfig) (webhooksetconfig.WebhookSecrets, error) {
	secrets, err := webhookconfig.BuildWebhookSecrets(&cfg, webhookconfig.GenerateWebhookSecret)
	if err != nil {
		return webhooksetconfig.WebhookSecrets{}, err
	}
	return webhooksetconfig.WebhookSecrets{Repos: secrets.Repos}, nil
}

func (c *setconfigSecretsClient) Write(k8sCtx webhooksetconfig.K8sContext, secrets webhooksetconfig.WebhookSecrets) error {
	s := &webhookconfig.Secrets{Repos: secrets.Repos}
	return webhookconfig.WriteWebhookSecretsAndLog(c.ctx, c.k8sClient, k8sCtx.Name, k8sCtx.Namespace, s, c.out)
}

type setconfigGitHubClient struct {
	ctx      context.Context
	ghClient github.GHClient
	out      *output.Client
}

func (c *setconfigGitHubClient) RegisterWebhooks(secrets webhooksetconfig.WebhookSecrets) {
	webhookconfig.RegisterAllGitHubWebhooks(c.ctx, c.ghClient, c.out, secrets.Repos)
}
