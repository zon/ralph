package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/logger"
)

// ConfigPulumiCmd configures Pulumi credentials for Argo Workflows
type ConfigPulumiCmd struct {
	Token     string `arg:"" help:"Pulumi access token" optional:""`
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
}

// Run executes the config pulumi command
func (c *ConfigPulumiCmd) Run() error {
	ctx := context.Background()

	c.printHeader()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	k8sCtx, err := resolveKubeContext(ctx, cfg, c.Context, c.Namespace)
	if err != nil {
		return err
	}

	token := c.Token
	if token == "" {
		token, err = c.promptForToken()
		if err != nil {
			return err
		}
	}

	if err := c.createK8sSecret(ctx, k8sCtx.Name, k8sCtx.Namespace, token); err != nil {
		return err
	}

	return nil
}

func (c *ConfigPulumiCmd) printHeader() {
	logger.Info("Configuring Pulumi credentials for Ralph remote execution...")
}

func (c *ConfigPulumiCmd) promptForToken() (string, error) {
	logger.Info("Enter your Pulumi access token.")
	logger.Info("You can get a token from: https://app.pulumi.com/account/tokens")

	token := os.Getenv("PULUMI_ACCESS_TOKEN")
	if token != "" {
		logger.Info("Using PULUMI_ACCESS_TOKEN from environment")
		return token, nil
	}

	fmt.Print("Pulumi access token: ")
	fmt.Scanln(&token)

	if token == "" {
		return "", fmt.Errorf("Pulumi access token is required")
	}

	return token, nil
}

func (c *ConfigPulumiCmd) createK8sSecret(ctx context.Context, kubeContext, namespace, token string) error {
	logger.Infof("Creating/updating Kubernetes secret '%s'...", k8s.PulumiSecretName)

	secretData := map[string]string{
		"PULUMI_ACCESS_TOKEN": token,
	}

	if err := k8s.CreateOrUpdateSecret(ctx, k8s.PulumiSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	logger.Successf("Secret '%s' created/updated successfully", k8s.PulumiSecretName)

	logger.Infof("Configuration complete! The secret '%s' is ready for use in namespace '%s'.", k8s.PulumiSecretName, namespace)
	return nil
}
