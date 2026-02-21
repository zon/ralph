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
	// GitSecretName is the name of the Kubernetes secret for git credentials
	GitSecretName = "git-credentials"
	// GitHubSecretName is the name of the Kubernetes secret for GitHub token
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

// IsGHCLIAvailable checks if the GitHub CLI (gh) is installed and available
func IsGHCLIAvailable(ctx context.Context) bool {
	_, err := exec.LookPath("gh")
	if err != nil {
		return false
	}

	// Check if gh is authenticated
	cmd := exec.CommandContext(ctx, "gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

// FindGitHubSSHKey searches for an SSH key by title on GitHub
// Returns the key ID if found, empty string if not found
func FindGitHubSSHKey(ctx context.Context, title string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "ssh-key", "list")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to list SSH keys: %w (stderr: %s)", err, stderr.String())
	}

	// Parse output to find key with matching title
	// Format: "TITLE KEY_TYPE KEY_DATA CREATED_DATE KEY_ID TYPE"
	// Example: "ralph-myrepo ssh-ed25519 AAAAC3... 2025-02-15T12:00:00Z 123456789 authentication"
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		// Skip warning lines and empty lines
		if strings.HasPrefix(line, "warning:") || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// First field is the title
		lineTitle := fields[0]
		if lineTitle == title {
			// Key ID is the second-to-last field (5th field in typical output)
			keyID := fields[len(fields)-2]
			return keyID, nil
		}
	}

	return "", nil
}

// DeleteGitHubSSHKey deletes an SSH key from GitHub by its ID
func DeleteGitHubSSHKey(ctx context.Context, keyID string) error {
	cmd := exec.CommandContext(ctx, "gh", "ssh-key", "delete", keyID, "--yes")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete SSH key: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// AddGitHubSSHKey adds an SSH public key to GitHub with the given title
func AddGitHubSSHKey(ctx context.Context, publicKey, title string) error {
	// Write public key to a temporary file
	// gh ssh-key add expects a file path or reads from stdin
	cmd := exec.CommandContext(ctx, "gh", "ssh-key", "add", "-", "-t", title)
	cmd.Stdin = strings.NewReader(publicKey)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add SSH key: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// GetGitHubRepo extracts the repository name and owner from git remote origin
// Returns: repoName, repoOwner, error
func GetGitHubRepo(ctx context.Context) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "--get", "remote.origin.url")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("failed to get remote.origin.url: %w (stderr: %s)", err, stderr.String())
	}

	remoteURL := strings.TrimSpace(stdout.String())
	if remoteURL == "" {
		return "", "", fmt.Errorf("remote.origin.url is empty")
	}

	// Parse GitHub URL
	// Formats:
	// - https://github.com/owner/repo.git
	// - git@github.com:owner/repo.git
	// - https://github.com/owner/repo
	// - git@github.com:owner/repo

	var repoPath string

	if strings.HasPrefix(remoteURL, "git@github.com:") {
		// SSH format: git@github.com:owner/repo.git
		repoPath = strings.TrimPrefix(remoteURL, "git@github.com:")
	} else if strings.Contains(remoteURL, "github.com/") {
		// HTTPS format: https://github.com/owner/repo.git
		parts := strings.Split(remoteURL, "github.com/")
		if len(parts) > 1 {
			repoPath = parts[1]
		}
	} else {
		return "", "", fmt.Errorf("not a GitHub repository URL: %s", remoteURL)
	}

	// Remove .git suffix if present
	repoPath = strings.TrimSuffix(repoPath, ".git")

	// Split into owner/repo
	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository path: %s", repoPath)
	}

	repoOwner := parts[0]
	repoName := parts[1]

	return repoName, repoOwner, nil
}
