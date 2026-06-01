package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
)

func TestResolveKubeContext(t *testing.T) {
	tests := []struct {
		name              string
		flagContext       string
		flagNamespace     string
		configContext     string
		configNamespace   string
		mockContext       string
		mockNamespace     string
		mockError         bool
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
			mockContext:       "kubectl-context",
			mockNamespace:     "kubectl-namespace",
			expectedContext:   "flag-context",
			expectedNamespace: "flag-namespace",
		},
		{
			name:              "flag namespace only, context from config",
			flagContext:       "",
			flagNamespace:     "flag-namespace",
			configContext:     "config-context",
			configNamespace:   "config-namespace",
			mockContext:       "kubectl-context",
			mockNamespace:     "kubectl-namespace",
			expectedContext:   "config-context",
			expectedNamespace: "flag-namespace",
		},
		{
			name:              "flag namespace only, context from mock",
			flagContext:       "",
			flagNamespace:     "flag-namespace",
			configContext:     "",
			configNamespace:   "",
			mockContext:       "kubectl-context",
			mockNamespace:     "kubectl-namespace",
			expectedContext:   "kubectl-context",
			expectedNamespace: "flag-namespace",
		},
		{
			name:              "config context and namespace used when flag is empty",
			flagContext:       "",
			flagNamespace:     "",
			configContext:     "config-context",
			configNamespace:   "config-namespace",
			mockContext:       "kubectl-context",
			mockNamespace:     "kubectl-namespace",
			expectedContext:   "config-context",
			expectedNamespace: "config-namespace",
		},
		{
			name:              "config namespace used when flag is empty, mock context",
			flagContext:       "",
			flagNamespace:     "",
			configContext:     "",
			configNamespace:   "config-namespace",
			mockContext:       "kubectl-context",
			mockNamespace:     "kubectl-namespace",
			expectedContext:   "kubectl-context",
			expectedNamespace: "config-namespace",
		},
		{
			name:              "mock context and namespace used when flags and config are empty",
			flagContext:       "",
			flagNamespace:     "",
			configContext:     "",
			configNamespace:   "",
			mockContext:       "kubectl-context",
			mockNamespace:     "kubectl-namespace",
			expectedContext:   "kubectl-context",
			expectedNamespace: "kubectl-namespace",
		},
		{
			name:              "mock namespace empty defaults to default",
			flagContext:       "",
			flagNamespace:     "",
			configContext:     "",
			configNamespace:   "",
			mockContext:       "kubectl-context",
			mockNamespace:     "",
			expectedContext:   "kubectl-context",
			expectedNamespace: "default",
		},
		{
			name:              "defaults to 'config' namespace when config file exists but no namespace set",
			flagContext:       "",
			flagNamespace:     "",
			configContext:     "config-context",
			configNamespace:   "",
			mockContext:       "kubectl-context",
			mockNamespace:     "kubectl-namespace",
			expectedContext:   "config-context",
			expectedNamespace: "config",
		},
		{
			name:        "mock context error returns error when no flag or config context",
			flagContext: "",
			flagNamespace: "",
			configContext: "",
			configNamespace: "",
			mockError:   true,
			expectError: true,
		},
		{
			name:        "error when .ralph directory is missing",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			dir := t.TempDir()
			t.Chdir(dir)

			if tt.name != "error when .ralph directory is missing" {
				ralphDir := filepath.Join(dir, ".ralph")
				err := os.MkdirAll(ralphDir, 0755)
				require.NoError(t, err)

				if tt.configContext != "" || tt.configNamespace != "" {
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
			}

			cfg, err := config.LoadConfig()
			if err != nil && !tt.expectError {
				require.NoError(t, err)
			}

			mockClient := &k8s.MockClient{}
			if tt.mockError {
				mockClient.GetCurrentContextFunc = func(ctx context.Context) (k8s.Context, error) {
					return k8s.Context{}, fmt.Errorf("mock error")
				}
			} else if tt.mockContext != "" {
				mockClient.GetCurrentContextFunc = func(ctx context.Context) (k8s.Context, error) {
					return k8s.Context{Name: tt.mockContext, Namespace: tt.mockNamespace}, nil
				}
			}

			if tt.expectError {
				if err != nil {
					require.Error(t, err)
					return
				}
				_, err = resolveKubeContext(ctx, mockClient, cfg, tt.flagContext, tt.flagNamespace)
				require.Error(t, err)
				return
			}

			k8sCtx, err := resolveKubeContext(ctx, mockClient, cfg, tt.flagContext, tt.flagNamespace)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedContext, k8sCtx.Name)
			assert.Equal(t, tt.expectedNamespace, k8sCtx.Namespace)
		})
	}
}

func TestResolveKubeContext_UpwardSearch(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	ralphDir := filepath.Join(dir, ".ralph")
	err := os.MkdirAll(ralphDir, 0755)
	require.NoError(t, err)

	configContent := "workflow:\n  namespace: parent-ns\n"
	err = os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	subDir := filepath.Join(dir, "subdir", "subsubdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	t.Chdir(subDir)

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	mockClient := &k8s.MockClient{}
	mockClient.GetCurrentContextFunc = func(ctx context.Context) (k8s.Context, error) {
		return k8s.Context{Name: "test-ctx", Namespace: "default"}, nil
	}

	k8sCtx, err := resolveKubeContext(ctx, mockClient, cfg, "", "")

	require.NoError(t, err)
	assert.Equal(t, "test-ctx", k8sCtx.Name)
	assert.Equal(t, "parent-ns", k8sCtx.Namespace)
}
