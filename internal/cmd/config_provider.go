package cmd

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/auth"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	configprovider "github.com/zon/ralph/internal/orchestration/configprovider"
	"github.com/zon/ralph/internal/prompt"
)

// ConfigProviderCmd configures an AI provider API key
type ConfigProviderCmd struct {
	Provider  string `arg:"" help:"AI provider name (anthropic, google, deepseek)"`
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
}

// Run executes the config provider command
func (c *ConfigProviderCmd) Run() error {
	switch c.Provider {
	case "anthropic", "google", "deepseek":
	default:
		return fmt.Errorf("unknown provider: %s", c.Provider)
	}

	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := resolveKubeContext(ctx, cfg, c.Context, c.Namespace)
	if err != nil {
		return err
	}

	runner := configprovider.New(
		&promptClientAdapter{},
		&authClientAdapter{},
		&k8sClientAdapter{},
	)

	return runner.Run(ctx, c.Provider, k8sCtx.Name, k8sCtx.Namespace)
}

type promptClientAdapter struct{}

func (a *promptClientAdapter) ProviderKey(provider string) (string, error) {
	return prompt.ProviderKey(provider)
}

type authClientAdapter struct{}

func (a *authClientAdapter) Load() (map[string]string, error) {
	return auth.Load()
}

func (a *authClientAdapter) Write(keys map[string]string) error {
	return auth.Write(keys)
}

type k8sClientAdapter struct{}

func (a *k8sClientAdapter) StoreProviderSecret(ctx context.Context, kubeContext, namespace string, keys map[string]string) error {
	return k8s.StoreProviderSecret(ctx, kubeContext, namespace, keys)
}
