package webhooksetconfig

import "github.com/zon/ralph/internal/webhookconfig"

type K8sContext struct {
	Name      string
	Namespace string
}

type ContextClient interface {
	Resolve(flagContext, flagNamespace string) (K8sContext, error)
}

type ConfigClient interface {
	Build(k8sCtx K8sContext, configPath string) webhookconfig.AppConfig
	Write(k8sCtx K8sContext, cfg webhookconfig.AppConfig) error
	Read(k8sCtx K8sContext) (webhookconfig.AppConfig, error)
}

type WebhookSecrets struct {
	Repos []webhookconfig.RepoSecret
}

type SecretsClient interface {
	Generate(cfg webhookconfig.AppConfig) (WebhookSecrets, error)
	Write(k8sCtx K8sContext, secrets WebhookSecrets) error
}

type GitHubClient interface {
	RegisterWebhooks(secrets WebhookSecrets)
}

type SetConfigCmd struct {
	Ctx     ContextClient
	Config  ConfigClient
	Secrets SecretsClient
	GitHub  GitHubClient
}

type Flags struct {
	Context    string
	Namespace  string
	ConfigPath string
}

func (c *SetConfigCmd) Run(flags Flags) error {
	k8sCtx, err := c.Ctx.Resolve(flags.Context, flags.Namespace)
	if err != nil {
		return err
	}

	appCfg := c.Config.Build(k8sCtx, flags.ConfigPath)

	if err := c.Config.Write(k8sCtx, appCfg); err != nil {
		return err
	}

	appCfg, err = c.Config.Read(k8sCtx)
	if err != nil {
		return err
	}

	secrets, err := c.Secrets.Generate(appCfg)
	if err != nil {
		return err
	}

	c.GitHub.RegisterWebhooks(secrets)

	return c.Secrets.Write(k8sCtx, secrets)
}
