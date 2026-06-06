# Set Config Orchestration

## Purpose

Run credential setup steps in order for ralph remote execution: resolve Kubernetes context, configure GitHub App credentials, configure OpenCode credentials.

## Orchestration

**Module:** `internal/orchestration/setconfig`

```go
type K8sContext struct {
	Name      string
	Namespace string
}

type ContextClient interface {
	Resolve(flagContext, flagNamespace string) (K8sContext, error)
}

type GitHubCredentialsClient interface {
	SecretExists(k8sCtx K8sContext) (bool, error)
	Validate(keyPath string) error
	Configure(k8sCtx K8sContext, keyPath string) error
}

type OpenCodeCredentialsClient interface {
	Configure(k8sCtx K8sContext) error
}

type SetConfigCmd struct {
	Ctx      ContextClient
	GitHub   GitHubCredentialsClient
	OpenCode OpenCodeCredentialsClient
}

type Flags struct {
	Context   string
	Namespace string
	GithubKey string
}

func (c *SetConfigCmd) Run(flags Flags) error {
	k8sCtx, err := c.Ctx.Resolve(flags.Context, flags.Namespace)
	if err != nil {
		return err
	}

	if err := c.configureGitHub(k8sCtx, flags.GithubKey); err != nil {
		return err
	}

	return c.OpenCode.Configure(k8sCtx)
}

func (c *SetConfigCmd) configureGitHub(k8sCtx K8sContext, keyPath string) error {
	if keyPath == "" {
		exists, err := c.GitHub.SecretExists(k8sCtx)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNoGitHubKey
		}
		return nil
	}

	if err := c.GitHub.Validate(keyPath); err != nil {
		return err
	}

	return c.GitHub.Configure(k8sCtx, keyPath)
}
```

### Helpers

- **`c.Ctx.Resolve(flagContext, flagNamespace)`** — resolves the target Kubernetes context and namespace from flags, falling back to the current kubeconfig context
- **`c.GitHub.SecretExists(k8sCtx)`** — reports whether the GitHub App credentials secret already exists in the target namespace
- **`c.GitHub.Validate(keyPath)`** — reads the private key at `keyPath` and validates it against the GitHub API
- **`c.GitHub.Configure(k8sCtx, keyPath)`** — reads the private key at `keyPath` and writes the GitHub App credentials secret to Kubernetes
- **`c.OpenCode.Configure(k8sCtx)`** — reads the local OpenCode `auth.json` and writes it as a Kubernetes secret

## Tests

**Module:** `internal/orchestration/setconfig`

```go
func TestRunConfiguresGitHubAndOpenCode(t *testing.T) {
	cmd := setconfig.withMocks()
	err := cmd.Run(flags.withKey())
	require.NoError(t, err)
	require.True(t, github.validateCalled())
	require.True(t, github.configureCalled())
	require.True(t, opencode.configureCalled())
}

func TestRunHaltsOnGitHubValidationFailure(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withGitHub(github.thatFailsValidation()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
	require.False(t, opencode.configureCalled())
}

func TestRunHaltsOnOpenCodeFailure(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withOpenCode(opencode.thatFails()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
}

func TestRunReusesExistingSecretWhenNoKeyProvided(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withGitHub(github.withExistingSecret()),
	)
	err := cmd.Run(flags.withoutKey())
	require.NoError(t, err)
	require.False(t, github.validateCalled())
	require.False(t, github.configureCalled())
}

func TestRunErrorsWhenNoKeyAndNoExistingSecret(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withGitHub(github.withNoExistingSecret()),
	)
	err := cmd.Run(flags.withoutKey())
	require.ErrorIs(t, err, setconfig.ErrNoGitHubKey)
}

func TestRunPropagatesContextResolutionFailure(t *testing.T) {
	cmd := setconfig.withMocks(
		setconfig.withContext(ctx.thatFails()),
	)
	err := cmd.Run(flags.withKey())
	require.Error(t, err)
	require.False(t, github.validateCalled())
}
```

### Helpers

- **`setconfig.withMocks(overrides...)`** — constructs a `SetConfigCmd` with default passing mock implementations; accepts option overrides
- **`setconfig.withGitHub(client)`** — override option that substitutes the GitHub credentials client
- **`setconfig.withOpenCode(client)`** — override option that substitutes the OpenCode credentials client
- **`setconfig.withContext(client)`** — override option that substitutes the context resolver
- **`flags.withKey()`** — returns a `Flags` value with a non-empty `GithubKey` path
- **`flags.withoutKey()`** — returns a `Flags` value with an empty `GithubKey`
- **`github.validateCalled()`** — reports whether `Validate` was called on the GitHub mock
- **`github.configureCalled()`** — reports whether `Configure` was called on the GitHub mock
- **`github.thatFailsValidation()`** — returns a GitHub mock whose `Validate` call returns an error
- **`github.withExistingSecret()`** — returns a GitHub mock whose `SecretExists` returns true
- **`github.withNoExistingSecret()`** — returns a GitHub mock whose `SecretExists` returns false
- **`opencode.configureCalled()`** — reports whether `Configure` was called on the OpenCode mock
- **`opencode.thatFails()`** — returns an OpenCode mock whose `Configure` call returns an error
- **`ctx.thatFails()`** — returns a context mock whose `Resolve` call returns an error
