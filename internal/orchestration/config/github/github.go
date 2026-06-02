package github

import (
	"context"
	"fmt"

	"github.com/zon/ralph/internal/config"
)

type ConfigLoader interface {
	Load() (*config.RalphConfig, error)
}

type GitHubClient interface {
	GetRepo(ctx context.Context) (owner, name string, err error)
	GenerateAppJWT(appID string, privateKeyPEM []byte) (string, error)
	GetInstallationID(ctx context.Context, jwtToken, owner, repo string) (int64, error)
	GetInstallationToken(ctx context.Context, jwtToken string, installationID int64) (string, error)
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
	ghClient     GitHubClient
	k8sClient    K8sClient
	log          Logger
}

func New(configLoader ConfigLoader, ghClient GitHubClient, k8sClient K8sClient, log Logger) *Cmd {
	return &Cmd{
		configLoader: configLoader,
		ghClient:     ghClient,
		k8sClient:    k8sClient,
		log:          log,
	}
}

func (c *Cmd) Run(ctx context.Context, privateKeyBytes []byte, flagContext, flagNamespace string) error {
	c.log.Info("Configuring GitHub App credentials for Ralph remote execution...")

	cfg, err := c.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := c.resolveKubeContext(ctx, cfg, flagContext, flagNamespace)
	if err != nil {
		return err
	}

	owner, name, err := c.ghClient.GetRepo(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect GitHub repository: %w", err)
	}

	if len(privateKeyBytes) == 0 {
		return fmt.Errorf("private key file is empty")
	}

	if err := c.validateCredentials(ctx, owner, name, privateKeyBytes); err != nil {
		return err
	}

	appID := config.DefaultAppID

	if err := c.createK8sSecret(ctx, k8sCtx.Name, k8sCtx.Namespace, appID, privateKeyBytes); err != nil {
		return err
	}

	c.log.Info("Note: GitHub App credentials are not tied to any user account.")

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

func (c *Cmd) validateCredentials(ctx context.Context, repoOwner, repoName string, privateKeyBytes []byte) error {
	c.log.Info("Validating credentials...")
	appID := config.DefaultAppID

	jwtToken, err := c.ghClient.GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to generate JWT for validation: %w", err)
	}

	installationID, err := c.ghClient.GetInstallationID(ctx, jwtToken, repoOwner, repoName)
	if err != nil {
		return fmt.Errorf("failed to get installation ID: %w", err)
	}

	_, err = c.ghClient.GetInstallationToken(ctx, jwtToken, installationID)
	if err != nil {
		return fmt.Errorf("failed to get installation token: %w", err)
	}

	c.log.Success("Credentials validated successfully")
	return nil
}

func (c *Cmd) createK8sSecret(ctx context.Context, kubeContext, namespace, appID string, privateKeyBytes []byte) error {
	const secretName = "github-credentials"

	c.log.Infof("Creating/updating Kubernetes secret '%s'...", secretName)

	secretData := map[string]string{
		"app-id":      appID,
		"private-key": string(privateKeyBytes),
	}

	if err := c.k8sClient.CreateOrUpdateSecret(ctx, secretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	c.log.Successf("Secret '%s' created/updated successfully", secretName)
	c.log.Infof("Configuration complete! The secret '%s' is ready for use in namespace '%s'.", secretName, namespace)

	return nil
}
