package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	githubpkg "github.com/zon/ralph/internal/github"
)

// resourcesFromRenderedWorkflow parses workflow YAML and returns the executor
// container resources block, or nil when the container carries none.
func resourcesFromRenderedWorkflow(t *testing.T, workflowYAML string) map[string]interface{} {
	t.Helper()

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "failed to parse workflow YAML")

	spec, ok := wfData["spec"].(map[string]interface{})
	require.True(t, ok, "spec must be a map")

	templates, ok := spec["templates"].([]interface{})
	require.True(t, ok, "templates must be a list")
	require.NotEmpty(t, templates, "templates must not be empty")

	tmpl, ok := templates[0].(map[string]interface{})
	require.True(t, ok, "template must be a map")

	container, ok := tmpl["container"].(map[string]interface{})
	require.True(t, ok, "container must be a map")

	resources, ok := container["resources"].(map[string]interface{})
	if !ok {
		return nil
	}
	return resources
}

// TestWorkflowRender_ResourcesFromConfig asserts the executor template carries
// the configured requests and limits unchanged in the rendered workflow.
func TestWorkflowRender_ResourcesFromConfig(t *testing.T) {
	cfg := &config.RalphConfig{
		DefaultBranch: "main",
		Workflow: config.WorkflowConfig{
			Resources: config.WorkflowResources{
				Requests: config.ResourceList{Memory: "1Gi", CPU: "500m"},
				Limits:   config.ResourceList{Memory: "2Gi", CPU: "1"},
			},
		},
	}
	ctx := &execcontext.Context{}

	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "main", "", false, "project.yaml", false, cfg, "")
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")

	assert.Equal(t, "1Gi", wf.Resources.Requests.Memory, "Workflow must carry the config memory request")
	assert.Equal(t, "500m", wf.Resources.Requests.CPU, "Workflow must carry the config cpu request")
	assert.Equal(t, "2Gi", wf.Resources.Limits.Memory, "Workflow must carry the config memory limit")
	assert.Equal(t, "1", wf.Resources.Limits.CPU, "Workflow must carry the config cpu limit")

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	resources := resourcesFromRenderedWorkflow(t, workflowYAML)
	require.NotNil(t, resources, "executor container must carry a resources block")

	requests, ok := resources["requests"].(map[string]interface{})
	require.True(t, ok, "resources must have a requests map")
	assert.Equal(t, "1Gi", requests["memory"])
	assert.Equal(t, "500m", requests["cpu"])

	limits, ok := resources["limits"].(map[string]interface{})
	require.True(t, ok, "resources must have a limits map")
	assert.Equal(t, "2Gi", limits["memory"])
	assert.Equal(t, "1", limits["cpu"])
}

// TestWorkflowRun_AppliesConfiguredResources asserts the workflow-run submit
// path loads resources from .ralph/config.yaml and applies them to the executor
// container in the rendered workflow.
func TestWorkflowRun_AppliesConfiguredResources(t *testing.T) {
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `workflow:
  resources:
    requests:
      memory: 1Gi
      cpu: 500m
    limits:
      memory: 2Gi
      cpu: "1"
`
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644))

	t.Chdir(tmpDir)

	ctx := &execcontext.Context{}
	wf, err := GenerateWorkflow(ctx, "test-project", "main", "test-project", "main", ".", false, false, "git@github.com:test/repo.git", "project.yaml")
	require.NoError(t, err, "GenerateWorkflow failed")

	assert.Equal(t, "1Gi", wf.Resources.Requests.Memory, "Workflow must carry the config memory request")
	assert.Equal(t, "500m", wf.Resources.Requests.CPU, "Workflow must carry the config cpu request")
	assert.Equal(t, "2Gi", wf.Resources.Limits.Memory, "Workflow must carry the config memory limit")
	assert.Equal(t, "1", wf.Resources.Limits.CPU, "Workflow must carry the config cpu limit")

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	resources := resourcesFromRenderedWorkflow(t, workflowYAML)
	require.NotNil(t, resources, "executor container must carry a resources block")

	requests, ok := resources["requests"].(map[string]interface{})
	require.True(t, ok, "resources must have a requests map")
	assert.Equal(t, "1Gi", requests["memory"])
	assert.Equal(t, "500m", requests["cpu"])

	limits, ok := resources["limits"].(map[string]interface{})
	require.True(t, ok, "resources must have a limits map")
	assert.Equal(t, "2Gi", limits["memory"])
	assert.Equal(t, "1", limits["cpu"])
}

// TestWorkflowRender_ResourcesOmitted asserts a config without resources still
// renders a valid workflow with no resources block on the container.
func TestWorkflowRender_ResourcesOmitted(t *testing.T) {
	cfg := &config.RalphConfig{DefaultBranch: "main"}
	ctx := &execcontext.Context{}

	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "main", "", false, "project.yaml", false, cfg, "")
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")

	assert.Empty(t, wf.Resources, "Workflow Resources must stay unset when the config omits them")

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	assert.NotContains(t, workflowYAML, "resources:", "workflow YAML must not carry a resources block when none are configured")
}

// TestWorkflowRender_PartialResources asserts only the configured quantities
// are rendered and unset entries are omitted.
func TestWorkflowRender_PartialResources(t *testing.T) {
	wf := &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		Resources: config.WorkflowResources{
			Requests: config.ResourceList{Memory: "512Mi"},
		},
	}

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	resources := resourcesFromRenderedWorkflow(t, workflowYAML)
	require.NotNil(t, resources, "executor container must carry a resources block")

	requests, ok := resources["requests"].(map[string]interface{})
	require.True(t, ok, "resources must have a requests map")
	assert.Equal(t, "512Mi", requests["memory"])
	assert.NotContains(t, requests, "cpu", "unset cpu request must be omitted")

	assert.NotContains(t, resources, "limits", "unset limits must be omitted")
}

// TestWorkflowSubmit_ResourcesRoundTrip asserts configured resources render
// unchanged in the workflow YAML handed to Argo at submission time.
func TestWorkflowSubmit_ResourcesRoundTrip(t *testing.T) {
	wf := &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		Resources: config.WorkflowResources{
			Requests: config.ResourceList{Memory: "1Gi", CPU: "500m"},
			Limits:   config.ResourceList{Memory: "2Gi", CPU: "1"},
		},
	}

	var submittedYAML string
	client := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submittedYAML = workflowYAML
			return "test-workflow", nil
		},
	}

	_, err := wf.Submit(context.Background(), client)
	require.NoError(t, err, "Submit failed")
	require.True(t, client.SubmitYAMLCalled, "SubmitYAML must be called")

	resources := resourcesFromRenderedWorkflow(t, submittedYAML)
	require.NotNil(t, resources, "executor container must carry a resources block")

	requests, ok := resources["requests"].(map[string]interface{})
	require.True(t, ok, "resources must have a requests map")
	assert.Equal(t, "1Gi", requests["memory"])
	assert.Equal(t, "500m", requests["cpu"])

	limits, ok := resources["limits"].(map[string]interface{})
	require.True(t, ok, "resources must have a limits map")
	assert.Equal(t, "2Gi", limits["memory"])
	assert.Equal(t, "1", limits["cpu"])
}

// TestWorkflowSubmit_ResourcesOmitted asserts a workflow without resources is
// still submitted with a valid YAML that carries no resources block.
func TestWorkflowSubmit_ResourcesOmitted(t *testing.T) {
	wf := &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
	}

	var submittedYAML string
	client := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submittedYAML = workflowYAML
			return "test-workflow", nil
		},
	}

	_, err := wf.Submit(context.Background(), client)
	require.NoError(t, err, "Submit failed")
	require.True(t, client.SubmitYAMLCalled, "SubmitYAML must be called")

	assert.NotContains(t, submittedYAML, "resources:", "submitted workflow YAML must not carry a resources block when none are configured")
}

// TestWorkflowSubmit_RejectsMalformedResources asserts malformed resource
// values are rejected with a descriptive error at submission time and the
// workflow is not handed to Argo.
func TestWorkflowSubmit_RejectsMalformedResources(t *testing.T) {
	tests := []struct {
		name      string
		resources config.WorkflowResources
		wantErr   string
	}{
		{
			name: "memory limit below request",
			resources: config.WorkflowResources{
				Requests: config.ResourceList{Memory: "2Gi"},
				Limits:   config.ResourceList{Memory: "1Gi"},
			},
			wantErr: `memory limit "1Gi" is below its request "2Gi"`,
		},
		{
			name: "cpu limit below request",
			resources: config.WorkflowResources{
				Requests: config.ResourceList{CPU: "2"},
				Limits:   config.ResourceList{CPU: "1"},
			},
			wantErr: `cpu limit "1" is below its request "2"`,
		},
		{
			name: "unknown memory request quantity",
			resources: config.WorkflowResources{
				Requests: config.ResourceList{Memory: "banana"},
			},
			wantErr: `invalid memory request "banana"`,
		},
		{
			name: "unknown cpu limit quantity",
			resources: config.WorkflowResources{
				Limits: config.ResourceList{CPU: "10bananas"},
			},
			wantErr: `invalid cpu limit "10bananas"`,
		},
		{
			name: "quantity with unknown suffix",
			resources: config.WorkflowResources{
				Requests: config.ResourceList{Memory: "1foo"},
			},
			wantErr: `invalid memory request "1foo"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := &Workflow{
				ProjectName:   "test-project",
				Repo:          githubpkg.MakeRepo("owner", "repo"),
				CloneBranch:   "main",
				ProjectBranch: "feature-branch",
				ProjectPath:   "project.yaml",
				Resources:     tt.resources,
			}

			client := &argo.MockClient{
				SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
					return "test-workflow", nil
				},
			}

			_, err := wf.Submit(context.Background(), client)
			require.Error(t, err, "Submit must reject malformed resources")
			assert.Contains(t, err.Error(), "workflow resources", "error must name the workflow resources")
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.False(t, client.SubmitYAMLCalled, "Submit must not call Argo for malformed resources")
		})
	}
}
