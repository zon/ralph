package k8s

import (
	"context"
	"fmt"
	"strings"
)

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

func (c *client) GetConfigMapData(ctx context.Context, name, namespace, kubeContext string) (string, error) {
	args := buildGetConfigMapDataArgs(name, namespace, kubeContext)

	stdout, err := runKubectl(ctx, nil, args...)
	if err != nil {
		return "", fmt.Errorf("failed to read configmap '%s' from namespace '%s': %w", name, namespace, err)
	}

	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		return "", fmt.Errorf("configmap '%s' exists but config.yaml key is empty", name)
	}

	return raw, nil
}

func buildGetConfigMapDataArgs(name, namespace, kubeContext string) []string {
	args := []string{"get", "configmap", name, "-n", namespace, "-o", `jsonpath={.data.config\.yaml}`}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	return args
}



