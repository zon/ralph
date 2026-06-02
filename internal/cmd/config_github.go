package cmd

import (
	"context"
	"os"

	"github.com/zon/ralph/internal/config"
	internalgithub "github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	ocfggithub "github.com/zon/ralph/internal/orchestration/config/github"
	"github.com/zon/ralph/internal/output"
)

type ConfigGithubCmd struct {
	PrivateKey string `arg:"" help:"Path to GitHub App private key (.pem file)" type:"existingfile"`
	Context    string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace  string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
	out        *output.Client
}

func (c *ConfigGithubCmd) Run() error {
	ctx := context.Background()

	if c.out == nil {
		c.out = output.NewClient(os.Stdout, os.Stderr, false)
	}

	privateKeyBytes, err := os.ReadFile(c.PrivateKey)
	if err != nil {
		return err
	}

	orchestrator := newConfigGithubOrchestrator(c.out)
	return orchestrator.Run(ctx, privateKeyBytes, c.Context, c.Namespace)
}

type configGithubConfigLoaderAdapter struct{}

func (a *configGithubConfigLoaderAdapter) Load() (*config.RalphConfig, error) {
	return config.LoadConfig()
}

type configGithubGitHubClientAdapter struct{}

func (a *configGithubGitHubClientAdapter) GetRepo(ctx context.Context) (string, string, error) {
	repo, err := internalgithub.GetRepo(ctx)
	if err != nil {
		return "", "", err
	}
	return repo.Owner, repo.Name, nil
}

func (a *configGithubGitHubClientAdapter) GenerateAppJWT(appID string, privateKeyPEM []byte) (string, error) {
	return internalgithub.GenerateAppJWT(appID, privateKeyPEM)
}

func (a *configGithubGitHubClientAdapter) GetInstallationID(ctx context.Context, jwtToken, owner, repo string) (int64, error) {
	return internalgithub.GetInstallationID(ctx, jwtToken, owner, repo)
}

func (a *configGithubGitHubClientAdapter) GetInstallationToken(ctx context.Context, jwtToken string, installationID int64) (string, error) {
	return internalgithub.GetInstallationToken(ctx, jwtToken, installationID)
}

type configGithubK8sClientAdapter struct{}

func (a *configGithubK8sClientAdapter) GetCurrentContext(ctx context.Context) (ocfggithub.K8sContext, error) {
	realClient := k8s.NewClient()
	k8sCtx, err := realClient.GetCurrentContext(ctx)
	if err != nil {
		return ocfggithub.K8sContext{}, err
	}
	return ocfggithub.K8sContext{Name: k8sCtx.Name, Namespace: k8sCtx.Namespace}, nil
}

func (a *configGithubK8sClientAdapter) CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	return k8s.NewClient().CreateOrUpdateSecret(ctx, name, namespace, kubeContext, data)
}

type configGithubLoggerAdapter struct {
	out *output.Client
}

func (a *configGithubLoggerAdapter) Info(msg string) {
	a.out.Info(msg)
}

func (a *configGithubLoggerAdapter) Infof(format string, args ...interface{}) {
	a.out.Infof(format, args...)
}

func (a *configGithubLoggerAdapter) Success(msg string) {
	a.out.Success(msg)
}

func (a *configGithubLoggerAdapter) Successf(format string, args ...interface{}) {
	a.out.Successf(format, args...)
}

func (a *configGithubLoggerAdapter) Debugf(format string, args ...interface{}) {
	a.out.Debugf(format, args...)
}

func newConfigGithubOrchestrator(out *output.Client) *ocfggithub.Cmd {
	return ocfggithub.New(
		&configGithubConfigLoaderAdapter{},
		&configGithubGitHubClientAdapter{},
		&configGithubK8sClientAdapter{},
		&configGithubLoggerAdapter{out: out},
	)
}
