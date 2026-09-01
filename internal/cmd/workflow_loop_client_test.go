package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/argo"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/output"
)

// TestLoopWorkflowClientNamespaceOverride asserts the --namespace and --context
// overrides on the loop command reach both the workflow submission and any
// followed logs, overriding the values in .ralph/config.yaml.
func TestLoopWorkflowClientNamespaceOverride(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("workflow:\n  context: config-context\n  namespace: config-namespace\n"), 0644))
	t.Chdir(dir)

	var submitCtx argo.K8sContext
	var logsCtx argo.K8sContext
	argoClient := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submitCtx = kubeCtx
			return "test-workflow", nil
		},
		LogsFunc: func(ctx argo.K8sContext, workflowName string, follow bool) error {
			logsCtx = ctx
			return nil
		},
	}

	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(io.Discard, io.Discard, false))
	ctx.SetRepoOwner("owner")
	ctx.SetRepoName("repo")
	ctx.SetKubeContext("override-context")
	ctx.SetKubeNamespace("override-namespace")

	adapter := &loopWorkflowClientAdapter{ctx: ctx, argoClient: argoClient, currentBranch: func() (string, error) { return "main", nil }}

	workflowName, err := adapter.Submit("fmt", []string{"run gofmt"}, 3)
	require.NoError(t, err, "Submit failed")
	require.Equal(t, "test-workflow", workflowName)

	t.Run("namespace and context override reach the submission", func(t *testing.T) {
		assert.Equal(t, "override-context", submitCtx.Name, "the context override wins over the config value for submission")
		assert.Equal(t, "override-namespace", submitCtx.Namespace, "the namespace override wins over the config value for submission")
	})

	t.Run("namespace and context override reach the followed logs", func(t *testing.T) {
		require.NoError(t, adapter.FollowLogs(workflowName))
		assert.Equal(t, "override-context", logsCtx.Name, "the context override is used for followed logs")
		assert.Equal(t, "override-namespace", logsCtx.Namespace, "the namespace override is used for followed logs")
	})
}

// TestLoopWorkflowClientNamespaceFallsBackToConfig asserts that when no
// --namespace or --context override is passed, the loop workflow uses the
// values from .ralph/config.yaml for the submission and any followed logs.
func TestLoopWorkflowClientNamespaceFallsBackToConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("workflow:\n  context: config-context\n  namespace: config-namespace\n"), 0644))
	t.Chdir(dir)

	var submitCtx argo.K8sContext
	var logsCtx argo.K8sContext
	argoClient := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submitCtx = kubeCtx
			return "test-workflow", nil
		},
		LogsFunc: func(ctx argo.K8sContext, workflowName string, follow bool) error {
			logsCtx = ctx
			return nil
		},
	}

	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(io.Discard, io.Discard, false))
	ctx.SetRepoOwner("owner")
	ctx.SetRepoName("repo")

	adapter := &loopWorkflowClientAdapter{ctx: ctx, argoClient: argoClient, currentBranch: func() (string, error) { return "main", nil }}

	workflowName, err := adapter.Submit("fmt", []string{"run gofmt"}, 3)
	require.NoError(t, err, "Submit failed")

	assert.Equal(t, "config-context", submitCtx.Name, "the submission falls back to the config context")
	assert.Equal(t, "config-namespace", submitCtx.Namespace, "the submission falls back to the config namespace")

	require.NoError(t, adapter.FollowLogs(workflowName))
	assert.Equal(t, "config-context", logsCtx.Name, "the followed logs fall back to the config context")
	assert.Equal(t, "config-namespace", logsCtx.Namespace, "the followed logs fall back to the config namespace")
}
