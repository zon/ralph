package k8s

import (
	"context"
	"fmt"
)

// buildConfigMapArgs builds the kubectl create configmap command arguments
func buildConfigMapArgs(name, namespace, kubeContext string, data map[string]string) []string {
	if namespace == "" {
		namespace = "default"
	}

	args := []string{"create", "configmap", name}

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

// buildConfigMapApplyArgs builds the kubectl apply command arguments for ConfigMap
func buildConfigMapApplyArgs(kubeContext string) []string {
	args := []string{"apply", "-f", "-"}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	return args
}

func (c *client) CreateOrUpdateConfigMap(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	stdout, err := runKubectl(ctx, nil, buildConfigMapArgs(name, namespace, kubeContext, data)...)
	if err != nil {
		return fmt.Errorf("failed to generate configmap YAML: %w", err)
	}

	_, err = runKubectl(ctx, stdout, buildConfigMapApplyArgs(kubeContext)...)
	if err != nil {
		return fmt.Errorf("failed to apply configmap: %w", err)
	}

	return nil
}
