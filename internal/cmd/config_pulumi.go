package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	ocfgpulumi "github.com/zon/ralph/internal/orchestration/config/pulumi"
	"github.com/zon/ralph/internal/output"
)

type ConfigPulumiCmd struct {
	Token     string `arg:"" help:"Pulumi access token" optional:""`
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
	out       *output.Client
}

func (c *ConfigPulumiCmd) Run() error {
	ctx := context.Background()

	if c.out == nil {
		c.out = output.NewClient(os.Stdout, os.Stderr, false)
	}

	orchestrator := newConfigPulumiOrchestrator(c.out)
	return orchestrator.Run(ctx, c.Token, c.Context, c.Namespace)
}

type configPulumiConfigLoaderAdapter struct{}

func (a *configPulumiConfigLoaderAdapter) Load() (*config.RalphConfig, error) {
	return config.LoadConfig()
}

type configPulumiEnvClientAdapter struct{}

func (a *configPulumiEnvClientAdapter) Getenv(key string) string {
	return os.Getenv(key)
}

func (a *configPulumiEnvClientAdapter) Prompt(promptMsg string) (string, error) {
	fmt.Print(promptMsg)
	var token string
	_, err := fmt.Scanln(&token)
	if err != nil {
		return "", err
	}
	return token, nil
}

type configPulumiK8sClientAdapter struct{}

func (a *configPulumiK8sClientAdapter) GetCurrentContext(ctx context.Context) (ocfgpulumi.K8sContext, error) {
	realClient := k8s.NewClient()
	k8sCtx, err := realClient.GetCurrentContext(ctx)
	if err != nil {
		return ocfgpulumi.K8sContext{}, err
	}
	return ocfgpulumi.K8sContext{Name: k8sCtx.Name, Namespace: k8sCtx.Namespace}, nil
}

func (a *configPulumiK8sClientAdapter) CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	return k8s.NewClient().CreateOrUpdateSecret(ctx, name, namespace, kubeContext, data)
}

type configPulumiLoggerAdapter struct {
	out *output.Client
}

func (a *configPulumiLoggerAdapter) Info(msg string) {
	a.out.Info(msg)
}

func (a *configPulumiLoggerAdapter) Infof(format string, args ...interface{}) {
	a.out.Infof(format, args...)
}

func (a *configPulumiLoggerAdapter) Success(msg string) {
	a.out.Success(msg)
}

func (a *configPulumiLoggerAdapter) Successf(format string, args ...interface{}) {
	a.out.Successf(format, args...)
}

func (a *configPulumiLoggerAdapter) Debugf(format string, args ...interface{}) {
	a.out.Debugf(format, args...)
}

func newConfigPulumiOrchestrator(out *output.Client) *ocfgpulumi.Cmd {
	return ocfgpulumi.New(
		&configPulumiConfigLoaderAdapter{},
		&configPulumiEnvClientAdapter{},
		&configPulumiK8sClientAdapter{},
		&configPulumiLoggerAdapter{out: out},
	)
}
