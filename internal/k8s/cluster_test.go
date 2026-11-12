package k8s

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInCluster(t *testing.T) {
	tests := []struct {
		name          string
		serviceHost   string
		kubeconfig    string
		defaultKube   bool
		wantInCluster bool
	}{
		{
			name:          "service host with no kubeconfig is in cluster",
			serviceHost:   "10.0.0.1",
			kubeconfig:    "",
			wantInCluster: true,
		},
		{
			name:          "no service host is not in cluster",
			serviceHost:   "",
			wantInCluster: false,
		},
		{
			name:          "service host with a kubeconfig is not in cluster",
			serviceHost:   "10.0.0.1",
			kubeconfig:    "kubeconfig",
			wantInCluster: false,
		},
		{
			name:          "service host with the default kubeconfig is not in cluster",
			serviceHost:   "10.0.0.1",
			defaultKube:   true,
			wantInCluster: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("KUBERNETES_SERVICE_HOST", tt.serviceHost)
			t.Setenv("KUBECONFIG", "")

			if tt.kubeconfig != "" {
				path := filepath.Join(home, tt.kubeconfig)
				require.NoError(t, os.WriteFile(path, []byte("apiVersion: v1\n"), 0644))
				t.Setenv("KUBECONFIG", path)
			}

			if tt.defaultKube {
				require.NoError(t, os.MkdirAll(filepath.Join(home, ".kube"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(home, ".kube", "config"), []byte("apiVersion: v1\n"), 0644))
			}

			assert.Equal(t, tt.wantInCluster, InCluster())
		})
	}
}

func TestPodNamespace(t *testing.T) {
	namespacePath := filepath.Join(t.TempDir(), "namespace")
	original := serviceAccountNamespacePath
	serviceAccountNamespacePath = namespacePath
	t.Cleanup(func() { serviceAccountNamespacePath = original })

	t.Run("environment namespace wins", func(t *testing.T) {
		t.Setenv("POD_NAMESPACE", "from-env")
		require.NoError(t, os.WriteFile(namespacePath, []byte("from-file\n"), 0644))

		namespace, err := PodNamespace()
		require.NoError(t, err)
		assert.Equal(t, "from-env", namespace)
	})

	t.Run("service account namespace is read when the environment is unset", func(t *testing.T) {
		t.Setenv("POD_NAMESPACE", "")
		require.NoError(t, os.WriteFile(namespacePath, []byte("from-file\n"), 0644))

		namespace, err := PodNamespace()
		require.NoError(t, err)
		assert.Equal(t, "from-file", namespace)
	})

	t.Run("an empty namespace file fails", func(t *testing.T) {
		t.Setenv("POD_NAMESPACE", "")
		require.NoError(t, os.WriteFile(namespacePath, []byte("  \n"), 0644))

		_, err := PodNamespace()
		require.Error(t, err)
	})

	t.Run("a missing namespace file fails", func(t *testing.T) {
		t.Setenv("POD_NAMESPACE", "")
		require.NoError(t, os.Remove(namespacePath))

		_, err := PodNamespace()
		require.Error(t, err)
	})
}
