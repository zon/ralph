package cmd

import (
	"context"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	ocfgopencode "github.com/zon/ralph/internal/orchestration/config/opencode"
	"github.com/zon/ralph/internal/output"
)

type ConfigOpencodeCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
	out       *output.Client
}

func (c *ConfigOpencodeCmd) Run() error {
	ctx := context.Background()

	if c.out == nil {
		c.out = output.NewClient(os.Stdout, os.Stderr, false)
	}

	orchestrator := newConfigOpencodeOrchestrator(c.out)
	return orchestrator.Run(ctx, c.Context, c.Namespace)
}

type configOpencodeConfigLoaderAdapter struct{}

func (a *configOpencodeConfigLoaderAdapter) Load() (*config.RalphConfig, error) {
	return config.LoadConfig()
}

type configOpencodeFsClientAdapter struct{}

func (a *configOpencodeFsClientAdapter) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (a *configOpencodeFsClientAdapter) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

type configOpencodeK8sClientAdapter struct{}

func (a *configOpencodeK8sClientAdapter) GetCurrentContext(ctx context.Context) (ocfgopencode.K8sContext, error) {
	realClient := k8s.NewClient()
	k8sCtx, err := realClient.GetCurrentContext(ctx)
	if err != nil {
		return ocfgopencode.K8sContext{}, err
	}
	return ocfgopencode.K8sContext{Name: k8sCtx.Name, Namespace: k8sCtx.Namespace}, nil
}

func (a *configOpencodeK8sClientAdapter) CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	return k8s.NewClient().CreateOrUpdateSecret(ctx, name, namespace, kubeContext, data)
}

type configOpencodeLoggerAdapter struct {
	out *output.Client
}

func (a *configOpencodeLoggerAdapter) Info(msg string) {
	a.out.Info(msg)
}

func (a *configOpencodeLoggerAdapter) Infof(format string, args ...interface{}) {
	a.out.Infof(format, args...)
}

func (a *configOpencodeLoggerAdapter) Success(msg string) {
	a.out.Success(msg)
}

func (a *configOpencodeLoggerAdapter) Successf(format string, args ...interface{}) {
	a.out.Successf(format, args...)
}

func (a *configOpencodeLoggerAdapter) Debugf(format string, args ...interface{}) {
	a.out.Debugf(format, args...)
}

func newConfigOpencodeOrchestrator(out *output.Client) *ocfgopencode.Cmd {
	return ocfgopencode.New(
		&configOpencodeConfigLoaderAdapter{},
		&configOpencodeFsClientAdapter{},
		&configOpencodeK8sClientAdapter{},
		&configOpencodeLoggerAdapter{out: out},
	)
}
