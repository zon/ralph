package workflow

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
)

func TestWorkflowOptionsInCluster(t *testing.T) {
	tests := []struct {
		name              string
		flagContext       string
		flagNamespace     string
		configContext     string
		configNamespace   string
		expectedContext   string
		expectedNamespace string
	}{
		{
			name:              "ignores the flag and config context and uses the pod namespace",
			flagContext:       "flag-context",
			configContext:     "config-context",
			expectedNamespace: "pod-namespace",
		},
		{
			name:              "namespace flag wins over config and the pod namespace",
			flagNamespace:     "flag-namespace",
			configNamespace:   "config-namespace",
			expectedNamespace: "flag-namespace",
		},
		{
			name:              "config namespace wins over the pod namespace",
			configNamespace:   "config-namespace",
			expectedNamespace: "config-namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
			t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing-kubeconfig"))
			t.Setenv("POD_NAMESPACE", "pod-namespace")

			ctx := execcontext.NewContext()
			ctx.SetKubeContext(tt.flagContext)
			ctx.SetKubeNamespace(tt.flagNamespace)

			cfg := &config.RalphConfig{
				Workflow: config.WorkflowConfig{
					Context:   tt.configContext,
					Namespace: tt.configNamespace,
				},
			}

			opts, err := workflowOptionsFromConfig(cfg, ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedContext, opts.KubeContext, "an in-cluster workflow ignores every context")
			assert.Equal(t, tt.expectedNamespace, opts.Namespace)
		})
	}
}
