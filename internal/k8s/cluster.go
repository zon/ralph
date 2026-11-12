package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	serviceHostEnv        = "KUBERNETES_SERVICE_HOST"
	podNamespaceEnv       = "POD_NAMESPACE"
	defaultKubeconfigPath = ".kube/config"
)

var serviceAccountNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// InCluster reports whether Ralph runs in a pod with no kubeconfig. Such a
// pod reaches its own cluster through its ServiceAccount instead of a context.
func InCluster() bool {
	if os.Getenv(serviceHostEnv) == "" {
		return false
	}
	return !hasKubeconfig()
}

// PodNamespace returns the namespace the pod runs in. POD_NAMESPACE wins when a
// deployment supplies it through the Downward API; otherwise Ralph reads the
// ServiceAccount namespace file.
func PodNamespace() (string, error) {
	if namespace := strings.TrimSpace(os.Getenv(podNamespaceEnv)); namespace != "" {
		return namespace, nil
	}

	data, err := os.ReadFile(serviceAccountNamespacePath)
	if err != nil {
		return "", fmt.Errorf("failed to read the pod namespace: %w", err)
	}

	namespace := strings.TrimSpace(string(data))
	if namespace == "" {
		return "", fmt.Errorf("the pod namespace is empty")
	}
	return namespace, nil
}

func hasKubeconfig() bool {
	if paths := os.Getenv("KUBECONFIG"); paths != "" {
		for _, path := range filepath.SplitList(paths) {
			if path == "" {
				continue
			}
			if _, err := os.Stat(path); err == nil {
				return true
			}
		}
		return false
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, defaultKubeconfigPath))
	return err == nil
}
