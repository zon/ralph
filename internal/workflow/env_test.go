package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	githubpkg "github.com/zon/ralph/internal/github"
)

// envFromRenderedWorkflow parses workflow YAML and returns the container env
// entries keyed by name.
func envFromRenderedWorkflow(t *testing.T, workflowYAML string) map[string]map[string]interface{} {
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

	env, ok := container["env"].([]interface{})
	require.True(t, ok, "env must be a list")

	envByName := make(map[string]map[string]interface{}, len(env))
	for _, e := range env {
		envVar, ok := e.(map[string]interface{})
		require.True(t, ok, "env entry must be a map")
		name, ok := envVar["name"].(string)
		require.True(t, ok, "name must be a string")
		envByName[name] = envVar
	}
	return envByName
}

// assertMixedEnvVars checks the literal and secret env entries rendered from
// a mixed workflow.
func assertMixedEnvVars(t *testing.T, envByName map[string]map[string]interface{}) {
	t.Helper()

	literalEntry, ok := envByName["LOG_LEVEL"]
	require.True(t, ok, "LOG_LEVEL env entry not found")
	assert.Equal(t, "debug", literalEntry["value"])
	assert.NotContains(t, literalEntry, "valueFrom")

	secretEntry, ok := envByName["API_KEY"]
	require.True(t, ok, "API_KEY env entry not found")
	valueFrom, ok := secretEntry["valueFrom"].(map[string]interface{})
	require.True(t, ok, "API_KEY valueFrom is not a map")
	secretKeyRef, ok := valueFrom["secretKeyRef"].(map[string]interface{})
	require.True(t, ok, "API_KEY valueFrom.secretKeyRef is not a map")
	assert.Equal(t, "my-secret", secretKeyRef["name"])
	assert.Equal(t, "api-key", secretKeyRef["key"])
	assert.NotContains(t, secretEntry, "value")
}

// mixedEnvWorkflow returns a workflow with a literal and a secret env var.
func mixedEnvWorkflow() *Workflow {
	return &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		Env: map[string]config.EnvVar{
			"LOG_LEVEL": {Value: "debug"},
			"API_KEY":   {SecretKeyRef: &config.SecretKeyRef{Name: "my-secret", Key: "api-key"}},
		},
	}
}

func TestWorkflowRender_MixedEnvVars(t *testing.T) {
	wf := mixedEnvWorkflow()

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "render failed")

	envByName := envFromRenderedWorkflow(t, workflowYAML)
	assertMixedEnvVars(t, envByName)

	repoURLEntry, ok := envByName["GIT_REPO_URL"]
	require.True(t, ok, "GIT_REPO_URL env entry not found")
	assert.Equal(t, "https://github.com/owner/repo.git", repoURLEntry["value"])
	assert.NotContains(t, repoURLEntry, "valueFrom")
}

func TestWorkflowSubmit_MixedEnvVars(t *testing.T) {
	wf := mixedEnvWorkflow()
	wf.KubeContext = "my-context"
	wf.Namespace = "my-namespace"

	var capturedYAML string
	var capturedKubeCtx argo.K8sContext
	client := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			capturedYAML = workflowYAML
			capturedKubeCtx = kubeCtx
			return "ralph-test-project-abc123", nil
		},
	}

	workflowName, err := wf.Submit(context.Background(), client)
	require.NoError(t, err, "submit failed")
	assert.Equal(t, "ralph-test-project-abc123", workflowName)
	assert.True(t, client.SubmitYAMLCalled, "SubmitYAML should be called")
	require.NotEmpty(t, capturedYAML, "SubmitYAML should receive the rendered workflow YAML")
	assert.Equal(t, "my-context", capturedKubeCtx.Name)
	assert.Equal(t, "my-namespace", capturedKubeCtx.Namespace)

	envByName := envFromRenderedWorkflow(t, capturedYAML)
	assertMixedEnvVars(t, envByName)

	builtins := map[string]string{
		"GIT_REPO_URL":             "https://github.com/owner/repo.git",
		"GITHUB_REPO_OWNER":        "owner",
		"GITHUB_REPO_NAME":         "repo",
		"GIT_BRANCH":               "main",
		"PROJECT_BRANCH":           "feature-branch",
		"PROJECT_PATH":             "{{workflow.parameters.project-path}}",
		"INSTRUCTIONS_MD":          "{{workflow.parameters.instructions-md}}",
		"RALPH_WORKFLOW_EXECUTION": "true",
		"RALPH_DEBUG_BRANCH":       "",
		"RALPH_VERBOSE":            "false",
		"RALPH_NO_SERVICES":        "false",
	}

	for name, want := range builtins {
		t.Run("builtin_"+name, func(t *testing.T) {
			entry, ok := envByName[name]
			require.True(t, ok, "%s env entry not found", name)
			assert.Equal(t, want, entry["value"], "%s value mismatch", name)
			assert.NotContains(t, entry, "valueFrom", "%s must be a plain value entry", name)
		})
	}

	require.Len(t, envByName, len(builtins)+2, "env must contain exactly %d built-ins plus 2 user env vars", len(builtins))
}
