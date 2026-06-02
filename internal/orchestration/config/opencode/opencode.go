package opencode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/config"
)

type ConfigLoader interface {
	Load() (*config.RalphConfig, error)
}

type FsClient interface {
	UserHomeDir() (string, error)
	ReadFile(name string) ([]byte, error)
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
	fsClient     FsClient
	k8sClient    K8sClient
	log          Logger
}

func New(configLoader ConfigLoader, fsClient FsClient, k8sClient K8sClient, log Logger) *Cmd {
	return &Cmd{
		configLoader: configLoader,
		fsClient:     fsClient,
		k8sClient:    k8sClient,
		log:          log,
	}
}

func (c *Cmd) Run(ctx context.Context, flagContext, flagNamespace string) error {
	c.log.Info("Configuring OpenCode credentials for Ralph remote execution...")

	cfg, err := c.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := c.resolveKubeContext(ctx, cfg, flagContext, flagNamespace)
	if err != nil {
		return err
	}

	authFileContent, err := c.readOpenCodeCredentials()
	if err != nil {
		return err
	}

	if err := c.createK8sSecret(ctx, k8sCtx.Name, k8sCtx.Namespace, authFileContent); err != nil {
		return err
	}

	return nil
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

func (c *Cmd) readOpenCodeCredentials() (string, error) {
	homeDir, err := c.fsClient.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	authFilePath := filepath.Join(homeDir, ".local/share/opencode/auth.json")
	c.log.Infof("Reading OpenCode credentials from: %s", authFilePath)

	authFileContent, err := c.fsClient.ReadFile(authFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("OpenCode auth.json not found at %s\n\nPlease ensure OpenCode is configured and the auth.json file exists.", authFilePath)
		}
		return "", fmt.Errorf("failed to read auth.json: %w", err)
	}

	if len(authFileContent) == 0 {
		return "", fmt.Errorf("auth.json is empty at %s", authFilePath)
	}

	c.log.Success("OpenCode credentials read successfully")
	return string(authFileContent), nil
}

func (c *Cmd) createK8sSecret(ctx context.Context, kubeContext, namespace, authFileContent string) error {
	const secretName = "opencode-credentials"

	c.log.Infof("Creating/updating Kubernetes secret '%s'...", secretName)

	secretData := map[string]string{
		"auth.json": authFileContent,
	}

	if err := c.k8sClient.CreateOrUpdateSecret(ctx, secretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	c.log.Successf("Secret '%s' created/updated successfully", secretName)
	c.log.Infof("Configuration complete! The secret '%s' is ready for use in namespace '%s'.", secretName, namespace)

	return nil
}
