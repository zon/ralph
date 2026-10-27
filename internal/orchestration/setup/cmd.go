package setup

import "errors"

var ErrNoGitHubKey = errors.New("--github-key is required when no existing GitHub credentials secret is found")

var ErrBothGitHubFlags = errors.New("--github-key and --github-token are mutually exclusive")

var ErrNoTargetedNamespace = errors.New("no namespace targeted; skipping namespace preparation")

var ErrNoNamespaceForCredentials = errors.New("writing credentials requires a targeted namespace: pass --namespace or set workflow.namespace in .ralph/config.yaml")

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
	ConfigureToken(k8sCtx K8sContext, token string) error
	TokenFromGHCli() string
	TokenFromEnv() string
	TokenFromSecret(k8sCtx K8sContext) (string, error)
	TokenWorkflowScope(token string) (present, known bool, err error)
}

type OpenCodeCredentialsClient interface {
	Configure(k8sCtx K8sContext) error
}

type LocalReadinessClient interface {
	ConfirmGitReady() error
	ConfirmGitHubCLIReady() error
	ConfirmOpenCodeReady() error
	ConfirmKubectlReady() error
	ConfirmArgoReady() error
}

type SetupCmd struct {
	Readiness LocalReadinessClient
	Ctx       ContextClient
	GitHub    GitHubCredentialsClient
	OpenCode  OpenCodeCredentialsClient
}

type Flags struct {
	Context     string
	Namespace   string
	GithubKey   string
	GithubToken string
}

// Result reports what namespace preparation changed.
type Result struct {
	GitHubSecretReused   bool
	WorkflowScopeKnown   bool
	WorkflowScopePresent bool
}

type githubSetup struct {
	reused bool
	token  string
}

func (c *SetupCmd) Run(flags Flags) (Result, error) {
	if flags.GithubKey != "" && flags.GithubToken != "" {
		return Result{}, ErrBothGitHubFlags
	}

	if err := c.confirmLocalReadiness(); err != nil {
		return Result{}, err
	}

	k8sCtx, err := c.Ctx.Resolve(flags.Context, flags.Namespace)
	if errors.Is(err, ErrNoTargetedNamespace) {
		if flags.GithubKey != "" || flags.GithubToken != "" {
			return Result{}, ErrNoNamespaceForCredentials
		}
		return Result{}, nil
	}
	if err != nil {
		return Result{}, err
	}

	if err := c.confirmClusterTooling(); err != nil {
		return Result{}, err
	}

	credentials, err := c.configureGitHub(k8sCtx, flags)
	if err != nil {
		return Result{}, err
	}

	result := c.checkWorkflowScope(credentials)

	if err := c.OpenCode.Configure(k8sCtx); err != nil {
		return Result{}, err
	}

	return result, nil
}

// checkWorkflowScope reports the workflow scope of the token setup wrote or
// reused. The check is advisory, so an unavailable answer leaves the scope
// unknown instead of failing preparation.
func (c *SetupCmd) checkWorkflowScope(credentials githubSetup) Result {
	result := Result{GitHubSecretReused: credentials.reused}
	if credentials.token == "" {
		return result
	}

	present, known, err := c.GitHub.TokenWorkflowScope(credentials.token)
	if err != nil {
		return result
	}

	result.WorkflowScopePresent = present
	result.WorkflowScopeKnown = known
	return result
}

func (c *SetupCmd) confirmLocalReadiness() error {
	if err := c.Readiness.ConfirmGitReady(); err != nil {
		return err
	}
	if err := c.Readiness.ConfirmGitHubCLIReady(); err != nil {
		return err
	}
	return c.Readiness.ConfirmOpenCodeReady()
}

func (c *SetupCmd) confirmClusterTooling() error {
	if err := c.Readiness.ConfirmKubectlReady(); err != nil {
		return err
	}
	return c.Readiness.ConfirmArgoReady()
}

func (c *SetupCmd) configureGitHub(k8sCtx K8sContext, flags Flags) (githubSetup, error) {
	if flags.GithubToken != "" {
		return githubSetup{token: flags.GithubToken}, c.GitHub.ConfigureToken(k8sCtx, flags.GithubToken)
	}

	if flags.GithubKey != "" {
		if err := c.GitHub.Validate(flags.GithubKey); err != nil {
			return githubSetup{}, err
		}

		return githubSetup{}, c.GitHub.Configure(k8sCtx, flags.GithubKey)
	}

	exists, err := c.GitHub.SecretExists(k8sCtx)
	if err != nil {
		return githubSetup{}, err
	}
	if exists {
		token, err := c.GitHub.TokenFromSecret(k8sCtx)
		if err != nil {
			return githubSetup{reused: true}, nil
		}
		return githubSetup{reused: true, token: token}, nil
	}

	return c.configureTokenFromFallback(k8sCtx)
}

func (c *SetupCmd) configureTokenFromFallback(k8sCtx K8sContext) (githubSetup, error) {
	token := c.GitHub.TokenFromGHCli()
	if token == "" {
		token = c.GitHub.TokenFromEnv()
	}
	if token == "" {
		return githubSetup{}, ErrNoGitHubKey
	}
	return githubSetup{token: token}, c.GitHub.ConfigureToken(k8sCtx, token)
}
