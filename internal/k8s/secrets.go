package k8s

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	// GitHubSecretName is the name of the Kubernetes secret for GitHub App credentials
	GitHubSecretName = "github-credentials"
	// OpenCodeSecretName is the name of the Kubernetes secret for OpenCode credentials
	OpenCodeSecretName = "opencode-credentials"
)

// GenerateSSHKeyPair generates an Ed25519 SSH key pair
// Returns: privateKeyPEM, publicKeyOpenSSH, error
func GenerateSSHKeyPair() (string, string, error) {
	// Generate Ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Convert private key to PEM format
	privKeyBytes, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privKeyPEM := pem.EncodeToMemory(privKeyBytes)
	if privKeyPEM == nil {
		return "", "", fmt.Errorf("failed to encode private key to PEM")
	}

	// Convert public key to OpenSSH format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	pubKeyOpenSSH := string(ssh.MarshalAuthorizedKey(sshPubKey))
	pubKeyOpenSSH = strings.TrimSpace(pubKeyOpenSSH)

	return string(privKeyPEM), pubKeyOpenSSH, nil
}

// CreateOrUpdateSecret creates or updates a Kubernetes secret
func CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found in PATH - please install kubectl")
	}

	// Use default namespace if not specified
	if namespace == "" {
		namespace = "default"
	}

	// Build kubectl command args
	args := []string{"create", "secret", "generic", name}

	// Add data as literal values
	for key, value := range data {
		args = append(args, fmt.Sprintf("--from-literal=%s=%s", key, value))
	}

	// Add namespace
	args = append(args, "-n", namespace)

	// Add context if specified
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}

	// Add dry-run and output to generate YAML, then apply to handle create/update
	args = append(args, "--dry-run=client", "-o", "yaml")

	// Generate the secret YAML
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate secret YAML: %w (stderr: %s)", err, stderr.String())
	}

	// Apply the secret (this handles both create and update)
	applyArgs := []string{"apply", "-f", "-"}
	if kubeContext != "" {
		applyArgs = append(applyArgs, "--context", kubeContext)
	}

	applyCmd := exec.CommandContext(ctx, "kubectl", applyArgs...)
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

	// Use default namespace if not specified
	if namespace == "" {
		namespace = "default"
	}

	// Build kubectl command args
	args := []string{"create", "configmap", name}

	// Add data as literal values
	for key, value := range data {
		args = append(args, fmt.Sprintf("--from-literal=%s=%s", key, value))
	}

	// Add namespace
	args = append(args, "-n", namespace)

	// Add context if specified
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}

	// Add dry-run and output to generate YAML, then apply to handle create/update
	args = append(args, "--dry-run=client", "-o", "yaml")

	// Generate the configmap YAML
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate configmap YAML: %w (stderr: %s)", err, stderr.String())
	}

	// Apply the configmap (this handles both create and update)
	applyArgs := []string{"apply", "-f", "-"}
	if kubeContext != "" {
		applyArgs = append(applyArgs, "--context", kubeContext)
	}

	applyCmd := exec.CommandContext(ctx, "kubectl", applyArgs...)
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
