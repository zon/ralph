package k8s

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const (
	GitHubSecretName   = "github-credentials"
	OpenCodeSecretName = "opencode-credentials"
)

// buildSecretManifest renders a Secret manifest in-process so the values never
// appear in kubectl's command line, where other local users could read them
// from the process table.
func buildSecretManifest(name, namespace string, data map[string]string) ([]byte, error) {
	if namespace == "" {
		namespace = "default"
	}

	encoded := make(map[string]string, len(data))
	for key, value := range data {
		encoded[key] = base64.StdEncoding.EncodeToString([]byte(value))
	}

	manifest := struct {
		APIVersion string `json:"apiVersion"`
		Kind       string `json:"kind"`
		Metadata   struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Type string            `json:"type"`
		Data map[string]string `json:"data"`
	}{
		APIVersion: "v1",
		Kind:       "Secret",
		Type:       "Opaque",
		Data:       encoded,
	}
	manifest.Metadata.Name = name
	manifest.Metadata.Namespace = namespace

	return json.Marshal(manifest)
}

func buildSecretApplyArgs(kubeContext string) []string {
	args := []string{"apply", "-f", "-"}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	return args
}

func (c *client) CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	manifest, err := buildSecretManifest(name, namespace, data)
	if err != nil {
		return fmt.Errorf("failed to generate secret manifest: %w", err)
	}

	_, err = runKubectl(ctx, bytes.NewBuffer(manifest), buildSecretApplyArgs(kubeContext)...)
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

func (c *client) GetSecretData(ctx context.Context, name, namespace, kubeContext string) (map[string]string, error) {
	stdout, err := runKubectl(ctx, nil, buildGetSecretDataArgs(name, namespace, kubeContext)...)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret '%s' from namespace '%s': %w", name, namespace, err)
	}

	return parseSecretData(stdout.Bytes())
}

func buildGetSecretDataArgs(name, namespace, kubeContext string) []string {
	args := []string{"get", "secret", name, "-n", namespace, "-o", "json"}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}
	return args
}

func parseSecretData(raw []byte) (map[string]string, error) {
	var document struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("failed to parse secret: %w", err)
	}

	data := make(map[string]string, len(document.Data))
	for key, value := range document.Data {
		decoded, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("failed to decode secret key '%s': %w", key, err)
		}
		data[key] = string(decoded)
	}

	return data, nil
}
