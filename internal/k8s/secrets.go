package k8s

import (
	"context"
	"fmt"
)

const (
	// GitHubSecretName is the name of the Kubernetes secret for GitHub App credentials
	GitHubSecretName = "github-credentials"
	// OpenCodeSecretName is the name of the Kubernetes secret for OpenCode credentials
	OpenCodeSecretName = "opencode-credentials"
	// PulumiSecretName is the name of the Kubernetes secret for Pulumi credentials
	PulumiSecretName = "pulumi-credentials"
)

// buildSecretArgs builds the kubectl create secret generic command arguments
func buildSecretArgs(name, namespace, kubeContext string, data map[string]string) []string {
	if namespace == "" {
		namespace = "default"
	}

	args := []string{"create", "secret", "generic", name}

	for key, value := range data {
		args = append(args, fmt.Sprintf("--from-literal=%s=%s", key, value))
	}

	args = append(args, "-n", namespace)

	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}

	args = append(args, "--dry-run=client", "-o", "yaml")

	return args
}

// buildSecretApplyArgs builds the kubectl apply command arguments
func buildSecretApplyArgs(kubeContext string) []string {
	args := []string{"apply", "-f", "-"}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	return args
}

// CreateOrUpdateSecret creates or updates a Kubernetes secret
func CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	// Generate the secret YAML
	stdout, err := runKubectl(ctx, nil, buildSecretArgs(name, namespace, kubeContext, data)...)
	if err != nil {
		return fmt.Errorf("failed to generate secret YAML: %w", err)
	}

	// Apply the secret (this handles both create and update)
	_, err = runKubectl(ctx, stdout, buildSecretApplyArgs(kubeContext)...)
	if err != nil {
		return fmt.Errorf("failed to apply secret: %w", err)
	}

	return nil
}
