package k8s

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
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

// CreateOrUpdateSecret creates or updates a Kubernetes secret
func CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found in PATH - please install kubectl")
	}

	// Generate the secret YAML
	cmd := exec.CommandContext(ctx, "kubectl", buildSecretArgs(name, namespace, kubeContext, data)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate secret YAML: %w (stderr: %s)", err, stderr.String())
	}

	// Apply the secret (this handles both create and update)
	applyCmd := exec.CommandContext(ctx, "kubectl", buildSecretApplyArgs(kubeContext)...)
	applyCmd.Stdin = &stdout
	var applyStderr bytes.Buffer
	applyCmd.Stderr = &applyStderr

	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("failed to apply secret: %w (stderr: %s)", err, applyStderr.String())
	}

	return nil
}

// CreateOrUpdateConfigMap creates or updates a Kubernetes ConfigMap
func CreateOrUpdateConfigMap(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found in PATH - please install kubectl")
	}

	// Generate the configmap YAML
	cmd := exec.CommandContext(ctx, "kubectl", buildConfigMapArgs(name, namespace, kubeContext, data)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate configmap YAML: %w (stderr: %s)", err, stderr.String())
	}

	// Apply the configmap (this handles both create and update)
	applyCmd := exec.CommandContext(ctx, "kubectl", buildConfigMapApplyArgs(kubeContext)...)
	applyCmd.Stdin = &stdout
	var applyStderr bytes.Buffer
	applyCmd.Stderr = &applyStderr

	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("failed to apply configmap: %w (stderr: %s)", err, applyStderr.String())
	}

	return nil
}

// GetCurrentContext gets the current Kubernetes context
func GetCurrentContext(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "config", "current-context")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current context: %w (stderr: %s)", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetNamespaceForContext gets the namespace for a given context
// Returns empty string if no namespace is configured (will use "default")
func GetNamespaceForContext(ctx context.Context, kubeContext string) (string, error) {
	args := []string{"config", "view", "-o", fmt.Sprintf("jsonpath='{.contexts[?(@.name==\"%s\")].context.namespace}'", kubeContext)}
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// It's ok if this fails - we'll use default namespace
		return "", nil
	}

	namespace := strings.Trim(strings.TrimSpace(stdout.String()), "'")
	return namespace, nil
}
