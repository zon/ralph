package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/config"
	githubpkg "github.com/zon/ralph/internal/github"
)

func TestWorkflowRender_MixedEnvVars(t *testing.T) {
	wf := &Workflow{
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

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	env := container["env"].([]interface{})

	var literalEntry, secretEntry, repoURLEntry map[string]interface{}
	for _, e := range env {
		envVar := e.(map[string]interface{})
		switch envVar["name"] {
		case "LOG_LEVEL":
			literalEntry = envVar
		case "API_KEY":
			secretEntry = envVar
		case "GIT_REPO_URL":
			repoURLEntry = envVar
		}
	}

	require.NotNil(t, literalEntry, "LOG_LEVEL env entry not found")
	assert.Equal(t, "debug", literalEntry["value"])
	assert.NotContains(t, literalEntry, "valueFrom")

	require.NotNil(t, secretEntry, "API_KEY env entry not found")
	valueFrom, ok := secretEntry["valueFrom"].(map[string]interface{})
	require.True(t, ok, "API_KEY valueFrom is not a map")
	secretKeyRef, ok := valueFrom["secretKeyRef"].(map[string]interface{})
	require.True(t, ok, "API_KEY valueFrom.secretKeyRef is not a map")
	assert.Equal(t, "my-secret", secretKeyRef["name"])
	assert.Equal(t, "api-key", secretKeyRef["key"])
	assert.NotContains(t, secretEntry, "value")

	require.NotNil(t, repoURLEntry, "GIT_REPO_URL env entry not found")
	assert.Equal(t, "https://github.com/owner/repo.git", repoURLEntry["value"])
	assert.NotContains(t, repoURLEntry, "valueFrom")
}
