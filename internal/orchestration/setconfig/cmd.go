package setconfig

import "errors"

var ErrNoGitHubKey = errors.New("--github-key is required when no existing GitHub credentials secret is found")

var ErrBothGitHubFlags = errors.New("--github-key and --github-token are mutually exclusive")

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
	Context     string
	Namespace   string
	GithubKey   string
	GithubToken string
}

func (c *SetConfigCmd) Run(flags Flags) error {
	if flags.GithubKey != "" && flags.GithubToken != "" {
		return ErrBothGitHubFlags
	}

	k8sCtx, err := c.Ctx.Resolve(flags.Context, flags.Namespace)
	if err != nil {
		return err
	}

	if err := c.configureGitHub(k8sCtx, flags); err != nil {
		return err
	}

	return c.OpenCode.Configure(k8sCtx)
}

func (c *SetConfigCmd) configureGitHub(k8sCtx K8sContext, flags Flags) error {
	if flags.GithubToken != "" {
		return c.GitHub.ConfigureToken(k8sCtx, flags.GithubToken)
	}

	if flags.GithubKey != "" {
		if err := c.GitHub.Validate(flags.GithubKey); err != nil {
			return err
		}

		return c.GitHub.Configure(k8sCtx, flags.GithubKey)
	}

	exists, err := c.GitHub.SecretExists(k8sCtx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return c.configureTokenFromFallback(k8sCtx)
}

func (c *SetConfigCmd) configureTokenFromFallback(k8sCtx K8sContext) error {
	token := c.GitHub.TokenFromGHCli()
	if token == "" {
		token = c.GitHub.TokenFromEnv()
	}
	if token == "" {
		return ErrNoGitHubKey
	}
	return c.GitHub.ConfigureToken(k8sCtx, token)
}
