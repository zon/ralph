package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/zon/ralph/internal/argo"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

// TestWorkflowClientNamespaceOverride asserts the --namespace and --context
// overrides on the run command reach both the workflow submission and any
// followed logs, overriding the values in .ralph/config.yaml.
func TestWorkflowClientNamespaceOverride(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
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
		LogsFunc: func(_ context.Context, ctx argo.K8sContext, workflowName string, follow bool) error {
			logsCtx = ctx
			return nil
		},
	}

	ctx := execcontext.NewContext()
	ctx.SetRepoOwner("owner")
	ctx.SetRepoName("repo")
	ctx.SetKubeContext("override-context")
	ctx.SetKubeNamespace("override-namespace")

	adapter := &workflowClientAdapter{ctx: ctx, argoClient: argoClient}

	workflowName, err := adapter.Submit(project.ForProjectInput(project.Any()), "main", "", "main", "")
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

// TestWorkflowClientNamespaceFallsBackToConfig asserts that when no --namespace
// or --context override is passed, the workflow uses the values from
// .ralph/config.yaml for the submission and any followed logs.
func TestWorkflowClientNamespaceFallsBackToConfig(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
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
		LogsFunc: func(_ context.Context, ctx argo.K8sContext, workflowName string, follow bool) error {
			logsCtx = ctx
			return nil
		},
	}

	ctx := execcontext.NewContext()
	ctx.SetRepoOwner("owner")
	ctx.SetRepoName("repo")

	adapter := &workflowClientAdapter{ctx: ctx, argoClient: argoClient}

	workflowName, err := adapter.Submit(project.ForProjectInput(project.Any()), "main", "", "main", "")
	require.NoError(t, err, "Submit failed")

	assert.Equal(t, "config-context", submitCtx.Name, "the submission falls back to the config context")
	assert.Equal(t, "config-namespace", submitCtx.Namespace, "the submission falls back to the config namespace")

	require.NoError(t, adapter.FollowLogs(workflowName))
	assert.Equal(t, "config-context", logsCtx.Name, "the followed logs fall back to the config context")
	assert.Equal(t, "config-namespace", logsCtx.Namespace, "the followed logs fall back to the config namespace")
}

// TestWorkflowClientSubmitMakesProjectPathRepoRelative asserts that the project
// path is made relative to the repository root even when the repository owner
// is already known, because the container runs in a clone where the absolute
// host path does not exist.
func TestWorkflowClientSubmitMakesProjectPathRepoRelative(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	testutil.MakeInitialCommit(t, dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "projects"), 0755))
	projectPath := filepath.Join(dir, "projects", "test.yaml")
	require.NoError(t, os.WriteFile(projectPath, []byte("slug: test-project\n"), 0644))
	t.Chdir(dir)

	var submittedYAML string
	argoClient := &argo.MockClient{
		SubmitYAMLFunc: func(_ context.Context, workflowYAML string, _ argo.K8sContext) (string, error) {
			submittedYAML = workflowYAML
			return "test-workflow", nil
		},
	}

	ctx := execcontext.NewContext()
	ctx.SetRepoOwner("owner")
	ctx.SetRepoName("repo")

	adapter := &workflowClientAdapter{ctx: ctx, argoClient: argoClient}
	input := project.ForProjectInput(&project.Project{Slug: "test-project", Path: projectPath})
	_, err := adapter.Submit(input, "main", "", "main", "")
	require.NoError(t, err, "Submit failed")

	var data map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(submittedYAML), &data))

	parameters := data["spec"].(map[string]interface{})["arguments"].(map[string]interface{})["parameters"].([]interface{})
	projectPathParam := ""
	for _, p := range parameters {
		param := p.(map[string]interface{})
		if param["name"] == "project-path" {
			projectPathParam, _ = param["value"].(string)
		}
	}
	assert.Equal(t, "projects/test.yaml", projectPathParam, "the project path should be relative to the repository root")
}
