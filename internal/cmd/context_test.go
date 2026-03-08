package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadContextAndNamespace(t *testing.T) {
	mockKubectl := func(dir, kubeContext, kubeNamespace string, causeError bool) string {
		ns := kubeNamespace
		script := `#!/bin/bash
if [ "$1" = "config" ] && [ "$2" = "current-context" ]; then
  echo "` + kubeContext + `"
elif [ "$1" = "config" ] && [ "$2" = "view" ]; then
  echo "` + ns + `"
else
  exit 1
fi
`
		if causeError {
			script = `#!/bin/bash
exit 1
`
		}
		scriptPath := filepath.Join(dir, "kubectl")
		os.WriteFile(scriptPath, []byte(script), 0755)
		return scriptPath
	}

	tests := []struct {
		name              string
		flagContext       string
		flagNamespace     string
		configContext     string
		configNamespace   string
		kubeContext       string
		kubeNamespace     string
		kubeContextError  bool
		expectedContext   string
		expectedNamespace string
		expectError       bool
	}{
		{
			name:              "flag context takes highest priority",
			flagContext:       "flag-context",
			flagNamespace:     "flag-namespace",
			configContext:     "config-context",
			configNamespace:   "config-namespace",
			kubeContext:       "kubectl-context",
			kubeNamespace:     "kubectl-namespace",
			expectedContext:   "flag-context",
			expectedNamespace: "flag-namespace",
		},
		{
			name:              "flag namespace only, context from config",
			flagContext:       "",
			flagNamespace:     "flag-namespace",
			configContext:     "config-context",
			configNamespace:   "config-namespace",
			kubeContext:       "kubectl-context",
			kubeNamespace:     "kubectl-namespace",
			expectedContext:   "config-context",
			expectedNamespace: "flag-namespace",
		},
		{
			name:              "flag namespace only, context from kubectl",
			flagContext:       "",
			flagNamespace:     "flag-namespace",
			configContext:     "",
			configNamespace:   "",
			kubeContext:       "kubectl-context",
			kubeNamespace:     "kubectl-namespace",
			expectedContext:   "kubectl-context",
			expectedNamespace: "flag-namespace",
		},
		{
			name:              "config context and namespace used when flag is empty",
			flagContext:       "",
			flagNamespace:     "",
			configContext:     "config-context",
			configNamespace:   "config-namespace",
			kubeContext:       "kubectl-context",
			kubeNamespace:     "kubectl-namespace",
			expectedContext:   "config-context",
			expectedNamespace: "config-namespace",
		},
		{
			name:              "config namespace used when flag is empty, kubectl context",
			flagContext:       "",
			flagNamespace:     "",
			configContext:     "",
			configNamespace:   "config-namespace",
			kubeContext:       "kubectl-context",
			kubeNamespace:     "kubectl-namespace",
			expectedContext:   "kubectl-context",
			expectedNamespace: "config-namespace",
		},
		{
			name:              "kubectl context and namespace used when flags and config are empty",
			flagContext:       "",
			flagNamespace:     "",
			configContext:     "",
			configNamespace:   "",
			kubeContext:       "kubectl-context",
			kubeNamespace:     "kubectl-namespace",
			expectedContext:   "kubectl-context",
			expectedNamespace: "kubectl-namespace",
		},
		{
			name:              "kubectl namespace empty defaults to default",
			flagContext:       "",
			flagNamespace:     "",
			configContext:     "",
			configNamespace:   "",
			kubeContext:       "kubectl-context",
			kubeNamespace:     "",
			expectedContext:   "kubectl-context",
			expectedNamespace: "default",
		},
		{
			name:             "kubectl context error returns error when no flag or config context",
			flagContext:      "",
			flagNamespace:    "",
			configContext:    "",
			configNamespace:  "",
			kubeContextError: true,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			dir := t.TempDir()
			t.Chdir(dir)

			kubectlPath := mockKubectl(dir, tt.kubeContext, tt.kubeNamespace, tt.kubeContextError)
			t.Setenv("PATH", filepath.Dir(kubectlPath)+":"+os.Getenv("PATH"))

			if tt.configContext != "" || tt.configNamespace != "" {
				ralphDir := filepath.Join(dir, ".ralph")
				err := os.MkdirAll(ralphDir, 0755)
				require.NoError(t, err)

				configContent := "workflow:\n"
				if tt.configContext != "" {
					configContent += "  context: " + tt.configContext + "\n"
				}
				if tt.configNamespace != "" {
					configContent += "  namespace: " + tt.configNamespace + "\n"
				}
				err = os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644)
				require.NoError(t, err)
			}

			kubeconfigPath := filepath.Join(dir, "kubeconfig")
			kubeconfigContent := "apiVersion: v1\nkind: Config\n"
			if tt.kubeContext != "" && !tt.kubeContextError {
				kubeconfigContent += "current-context: " + tt.kubeContext + "\n"
			}
			if tt.kubeContextError {
				kubeconfigContent = "invalid yaml"
			} else {
				kubeconfigContent += "contexts:\n"
				if tt.kubeContext != "" {
					kubeconfigContent += "- name: " + tt.kubeContext + "\n"
					if tt.kubeNamespace != "" {
						kubeconfigContent += "  context:\n"
						kubeconfigContent += "    namespace: " + tt.kubeNamespace + "\n"
					}
				}
			}
			err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644)
			require.NoError(t, err)

			t.Setenv("KUBECONFIG", kubeconfigPath)

			kubeContext, namespace, err := loadContextAndNamespace(ctx, tt.flagContext, tt.flagNamespace)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedContext, kubeContext)
			assert.Equal(t, tt.expectedNamespace, namespace)
		})
	}
}
