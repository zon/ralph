# Webhook Set Config Orchestration

## Purpose

Provision all Kubernetes resources required for the ralph-webhook service: build and write the webhook-config ConfigMap, generate per-repo HMAC secrets, register GitHub webhooks, and write the webhook-secrets Secret.

## Orchestration

**Module:** `internal/orchestration/webhooksetconfig`

```go
type K8sContext struct {
	Name      string
	Namespace string
}

type ContextClient interface {
	Resolve(flagContext, flagNamespace string) (K8sContext, error)
}

type ConfigClient interface {
	Build(k8sCtx K8sContext, configPath string) AppConfig
	Write(k8sCtx K8sContext, cfg AppConfig) error
	Read(k8sCtx K8sContext) (AppConfig, error)
}

type SecretsClient interface {
	Generate(cfg AppConfig) (WebhookSecrets, error)
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
```

### Helpers

- **`c.Ctx.Resolve(flagContext, flagNamespace)`** — resolves the target Kubernetes context and namespace from flags, defaulting to `ralph-webhook` namespace
- **`c.Config.Build(k8sCtx, configPath)`** — reads the existing ConfigMap from Kubernetes as the base, merges any partial config file at `configPath`, and fills remaining fields via auto-detection (repo owner, name, namespace, collaborators)
- **`c.Config.Write(k8sCtx, cfg)`** — writes the app config to the `webhook-config` ConfigMap in the target namespace
- **`c.Config.Read(k8sCtx)`** — reads the current `webhook-config` ConfigMap from Kubernetes
- **`c.Secrets.Generate(cfg)`** — generates a per-repo HMAC secret for each repo in the config
- **`c.GitHub.RegisterWebhooks(secrets)`** — registers GitHub webhooks for each repo; individual failures emit warnings and do not halt execution
- **`c.Secrets.Write(k8sCtx, secrets)`** — writes all per-repo secrets to the `webhook-secrets` Kubernetes Secret

## Tests

**Module:** `internal/orchestration/webhooksetconfig`

```go
func TestRunWritesConfigThenSecrets(t *testing.T) {
	cmd := webhooksetconfig.withMocks()
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, config.writeCalled())
	require.True(t, secrets.writeCalled())
}

func TestRunHaltsOnConfigWriteFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withConfig(config.thatFailsWrite()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, secrets.generateCalled())
	require.False(t, secrets.writeCalled())
}

func TestRunContinuesAfterWebhookRegistrationFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withGitHub(github.thatFailsRegistration()),
	)
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, secrets.writeCalled())
}

func TestRunPropagatesContextResolutionFailure(t *testing.T) {
	cmd := webhooksetconfig.withMocks(
		webhooksetconfig.withContext(ctx.thatFails()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, config.writeCalled())
}
```

### Helpers

- **`webhooksetconfig.withMocks(overrides...)`** — constructs a `SetConfigCmd` with default passing mock implementations; accepts option overrides
- **`webhooksetconfig.withConfig(client)`** — override option that substitutes the config client
- **`webhooksetconfig.withGitHub(client)`** — override option that substitutes the GitHub client
- **`webhooksetconfig.withContext(client)`** — override option that substitutes the context resolver
- **`flags.any()`** — returns a valid `Flags` value suitable for most tests
- **`config.writeCalled()`** — reports whether `Write` was called on the config mock
- **`config.thatFailsWrite()`** — returns a config mock whose `Write` call returns an error
- **`secrets.generateCalled()`** — reports whether `Generate` was called on the secrets mock
- **`secrets.writeCalled()`** — reports whether `Write` was called on the secrets mock
- **`github.thatFailsRegistration()`** — returns a GitHub mock that records webhook registration failures without propagating them
- **`ctx.thatFails()`** — returns a context mock whose `Resolve` call returns an error
