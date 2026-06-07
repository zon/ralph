package main

import (
	"context"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	webhooksetconfig "github.com/zon/ralph/internal/orchestration/webhooksetconfig"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/provisioning"
	"github.com/zon/ralph/internal/webhookconfig"
)

type SetCmd struct {
	Config SetConfigCmd `cmd:"" help:"Set webhook configuration and secrets"`
}

type SetConfigCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use" default:"ralph-webhook"`
	Config    string `help:"Path to a partial AppConfig YAML file to use as a starting point" type:"path" optional:""`
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
	kubeCtx, err := provisioning.GetKubeContext(c.ctx, c.k8sClient, flagContext)
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
	base, err := provisioning.ReadWebhookConfigFromK8s(c.ctx, k8sCtx.Namespace, k8sCtx.Name)
	if err != nil {
		c.out.Warnf("Could not read existing configmap '%s': %v (starting from scratch)", provisioning.WebhookConfigMapName, err)
		base = nil
	}

	var updates *webhookconfig.AppConfig
	if configPath != "" {
		loaded, err := webhookconfig.LoadAppConfig(configPath)
		if err != nil {
			c.out.Warnf("Failed to load partial config: %v (ignoring)", err)
		} else {
			updates = loaded
		}
	}

	return provisioning.BuildWebhookAppConfig(c.ctx, c.out, base, updates, "", "", "", c.ghClient)
}

func (c *setconfigCfgClient) Write(k8sCtx webhooksetconfig.K8sContext, cfg webhookconfig.AppConfig) error {
	return provisioning.WriteWebhookConfigMap(c.ctx, c.k8sClient, k8sCtx.Name, k8sCtx.Namespace, cfg)
}

func (c *setconfigCfgClient) Read(k8sCtx webhooksetconfig.K8sContext) (webhookconfig.AppConfig, error) {
	cfg, err := provisioning.ReadWebhookConfigFromK8s(c.ctx, k8sCtx.Namespace, k8sCtx.Name)
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
	secrets, err := provisioning.BuildWebhookSecrets(&cfg, provisioning.GenerateWebhookSecret)
	if err != nil {
		return webhooksetconfig.WebhookSecrets{}, err
	}
	return webhooksetconfig.WebhookSecrets{Repos: secrets.Repos}, nil
}

func (c *setconfigSecretsClient) Write(k8sCtx webhooksetconfig.K8sContext, secrets webhooksetconfig.WebhookSecrets) error {
	s := &webhookconfig.Secrets{Repos: secrets.Repos}
	if err := provisioning.WriteWebhookSecrets(c.ctx, c.k8sClient, k8sCtx.Name, k8sCtx.Namespace, s); err != nil {
		return err
	}
	c.out.Successf("Secret '%s' created/updated in namespace '%s'", provisioning.WebhookSecretsSecretName, k8sCtx.Namespace)
	return nil
}

type setconfigGitHubClient struct {
	ctx      context.Context
	ghClient github.GHClient
	out      *output.Client
}

func (c *setconfigGitHubClient) RegisterWebhooks(secrets webhooksetconfig.WebhookSecrets) {
	webhookURL := fmt.Sprintf("https://%s/webhook", provisioning.WebhookIngressHostname)
	c.out.Infof("Registering webhooks at %s...", webhookURL)
	for _, rs := range secrets.Repos {
		c.out.Infof("Registering webhook for %s/%s...", rs.Owner, rs.Name)
		if err := provisioning.RegisterGitHubWebhook(c.ctx, c.ghClient, rs.Owner, rs.Name, webhookURL, rs.WebhookSecret); err != nil {
			c.out.Warnf("Failed to register webhook for %s/%s: %v", rs.Owner, rs.Name, err)
		} else {
			c.out.Successf("Webhook registered for %s/%s", rs.Owner, rs.Name)
		}
	}
	c.out.Info("")
}
