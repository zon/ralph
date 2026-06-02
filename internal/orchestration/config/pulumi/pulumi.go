package pulumi

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/config"
)

type ConfigLoader interface {
	Load() (*config.RalphConfig, error)
}

type EnvClient interface {
	Getenv(key string) string
	Prompt(promptMsg string) (string, error)
}

type K8sContext struct {
	Name      string
	Namespace string
}

type K8sClient interface {
	GetCurrentContext(ctx context.Context) (K8sContext, error)
	CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
}

type Logger interface {
	Info(msg string)
	Infof(format string, args ...interface{})
	Success(msg string)
	Successf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type Cmd struct {
	configLoader ConfigLoader
	envClient    EnvClient
	k8sClient    K8sClient
	log          Logger
}

func New(configLoader ConfigLoader, envClient EnvClient, k8sClient K8sClient, log Logger) *Cmd {
	return &Cmd{
		configLoader: configLoader,
		envClient:    envClient,
		k8sClient:    k8sClient,
		log:          log,
	}
}

func (c *Cmd) Run(ctx context.Context, tokenFlag, flagContext, flagNamespace string) error {
	c.log.Info("Configuring Pulumi credentials for Ralph remote execution...")

	cfg, err := c.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := c.resolveKubeContext(ctx, cfg, flagContext, flagNamespace)
	if err != nil {
		return err
	}

	token, err := c.resolveToken(tokenFlag)
	if err != nil {
		return err
	}

	if err := c.createK8sSecret(ctx, k8sCtx.Name, k8sCtx.Namespace, token); err != nil {
		return err
	}

	return nil
}

func (c *Cmd) resolveToken(tokenFlag string) (string, error) {
	if tokenFlag != "" {
		c.log.Debugf("Using Pulumi token from flag")
		return tokenFlag, nil
	}

	token := c.envClient.Getenv("PULUMI_ACCESS_TOKEN")
	if token != "" {
		c.log.Info("Using PULUMI_ACCESS_TOKEN from environment")
		return token, nil
	}

	c.log.Info("Enter your Pulumi access token.")
	c.log.Info("You can get a token from: https://app.pulumi.com/account/tokens")

	token, err := c.envClient.Prompt("Pulumi access token: ")
	if err != nil {
		return "", err
	}

	if token == "" {
		return "", fmt.Errorf("Pulumi access token is required")
	}

	return token, nil
}

func (c *Cmd) resolveKubeContext(ctx context.Context, ralphConfig *config.RalphConfig, flagContext, flagNamespace string) (K8sContext, error) {
	var k8sCtx K8sContext

	if flagContext != "" {
		c.log.Debugf("Using Kubernetes context: %s", flagContext)
		k8sCtx.Name = flagContext
	} else if ralphConfig != nil && ralphConfig.Workflow.Context != "" {
		c.log.Debugf("Using context from .ralph/config.yaml: %s", ralphConfig.Workflow.Context)
		k8sCtx.Name = ralphConfig.Workflow.Context
	} else {
		current, err := c.k8sClient.GetCurrentContext(ctx)
		if err != nil {
			return K8sContext{}, fmt.Errorf("failed to get current Kubernetes context: %w\n\nMake sure kubectl is installed and configured.", err)
		}
		c.log.Debugf("Using current Kubernetes context: %s", current.Name)
		k8sCtx.Name = current.Name
		k8sCtx.Namespace = current.Namespace
	}

	if flagNamespace != "" {
		c.log.Debugf("Using namespace: %s", flagNamespace)
		k8sCtx.Namespace = flagNamespace
	} else if ralphConfig != nil && ralphConfig.Workflow.Namespace != "" {
		c.log.Debugf("Using namespace from .ralph/config.yaml: %s", ralphConfig.Workflow.Namespace)
		k8sCtx.Namespace = ralphConfig.Workflow.Namespace
	} else if ralphConfig != nil && ralphConfig.ConfigPath != "" {
		c.log.Debugf("Using default namespace: %s (config found)", "config")
		k8sCtx.Namespace = "config"
	}

	if k8sCtx.Namespace == "" {
		c.log.Debugf("Using namespace: %s (default)", "default")
		k8sCtx.Namespace = "default"
	}

	return k8sCtx, nil
}

func (c *Cmd) createK8sSecret(ctx context.Context, kubeContext, namespace, token string) error {
	const secretName = "pulumi-credentials"

	c.log.Infof("Creating/updating Kubernetes secret '%s'...", secretName)

	secretData := map[string]string{
		"PULUMI_ACCESS_TOKEN": token,
	}

	if err := c.k8sClient.CreateOrUpdateSecret(ctx, secretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	c.log.Successf("Secret '%s' created/updated successfully", secretName)
	c.log.Infof("Configuration complete! The secret '%s' is ready for use in namespace '%s'.", secretName, namespace)

	return nil
}
