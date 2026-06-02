package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/output"
)

// ConfigPulumiCmd configures Pulumi credentials for Argo Workflows
type ConfigPulumiCmd struct {
	Token     string `arg:"" help:"Pulumi access token" optional:""`
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
	out       *output.Client
}

// Run executes the config pulumi command
func (c *ConfigPulumiCmd) Run() error {
	ctx := context.Background()

	if c.out == nil {
		c.out = output.NewClient(os.Stdout, os.Stderr, false)
	}

	c.printHeader()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := k8s.NewClient()

	k8sCtx, err := resolveKubeContext(ctx, client, cfg, c.out, c.Context, c.Namespace)
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

	if err := c.createK8sSecret(ctx, client, k8sCtx.Name, k8sCtx.Namespace, token); err != nil {
		return err
	}

	return nil
}

func (c *ConfigPulumiCmd) printHeader() {
	c.out.Info("Configuring Pulumi credentials for Ralph remote execution...")
}

func (c *ConfigPulumiCmd) promptForToken() (string, error) {
	c.out.Info("Enter your Pulumi access token.")
	c.out.Info("You can get a token from: https://app.pulumi.com/account/tokens")

	token := os.Getenv("PULUMI_ACCESS_TOKEN")
	if token != "" {
		c.out.Info("Using PULUMI_ACCESS_TOKEN from environment")
		return token, nil
	}

	fmt.Print("Pulumi access token: ")
	fmt.Scanln(&token)

	if token == "" {
		return "", fmt.Errorf("Pulumi access token is required")
	}

	return token, nil
}

func (c *ConfigPulumiCmd) createK8sSecret(ctx context.Context, client k8s.Client, kubeContext, namespace, token string) error {
	c.out.Infof("Creating/updating Kubernetes secret '%s'...", k8s.PulumiSecretName)

	secretData := map[string]string{
		"PULUMI_ACCESS_TOKEN": token,
	}

	if err := client.CreateOrUpdateSecret(ctx, k8s.PulumiSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	c.out.Successf("Secret '%s' created/updated successfully", k8s.PulumiSecretName)

	c.out.Infof("Configuration complete! The secret '%s' is ready for use in namespace '%s'.", k8s.PulumiSecretName, namespace)
	return nil
}
