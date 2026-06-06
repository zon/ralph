package setconfig

import "errors"

var ErrNoGitHubKey = errors.New("--github-key is required when no existing GitHub credentials secret is found")

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
