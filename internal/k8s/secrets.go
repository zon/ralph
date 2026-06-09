package k8s

import (
	"bytes"
	"context"
	"fmt"
)

const (
	GitHubSecretName   = "github-credentials"
	OpenCodeSecretName = "opencode-credentials"
)

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

func buildSecretApplyArgs(kubeContext string) []string {
	args := []string{"apply", "-f", "-"}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	return args
}

func (c *client) CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	stdout, err := runKubectl(ctx, nil, buildSecretArgs(name, namespace, kubeContext, data)...)
	if err != nil {
		return fmt.Errorf("failed to generate secret YAML: %w", err)
	}

	_, err = runKubectl(ctx, stdout, buildSecretApplyArgs(kubeContext)...)
	if err != nil {
		return fmt.Errorf("failed to apply secret: %w", err)
	}

	return nil
}

func (c *client) SecretExists(ctx context.Context, name, namespace, kubeContext string) (bool, error) {
	args := []string{"get", "secret", name, "-n", namespace}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	_, err := runKubectl(ctx, nil, args...)
	if err != nil {
		if bytes.Contains([]byte(err.Error()), []byte("not found")) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
