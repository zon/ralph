package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestContainerBuildWorkflow(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "container-build.yaml")
	data, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "workflow file should exist at .github/workflows/container-build.yaml")

	var wf map[string]interface{}
	err = yaml.Unmarshal(data, &wf)
	require.NoError(t, err, "workflow should be valid YAML")

	require.Equal(t, "Container Build", wf["name"], "workflow should have correct name")

	on, ok := wf["on"].(map[string]interface{})
	require.True(t, ok, "workflow should have 'on' trigger")

	push, ok := on["push"].(map[string]interface{})
	require.True(t, ok, "workflow should trigger on push")

	branches, ok := push["branches"].([]interface{})
	require.True(t, ok, "push trigger should have branches")
	require.Contains(t, branches, "main", "workflow should trigger on push to main")

	perms, ok := wf["permissions"].(map[string]interface{})
	require.True(t, ok, "workflow should have permissions")
	require.Equal(t, "write", perms["packages"], "workflow should have packages write permission")

	jobs, ok := wf["jobs"].(map[string]interface{})
	require.True(t, ok, "workflow should have jobs")

	buildJob, ok := jobs["build"].(map[string]interface{})
	require.True(t, ok, "workflow should have a 'build' job")

	steps, ok := buildJob["steps"].([]interface{})
	require.True(t, ok, "build job should have steps")

	stepNames := make([]string, len(steps))
	for i, s := range steps {
		step := s.(map[string]interface{})
		if name, ok := step["name"].(string); ok {
			stepNames[i] = name
		}
	}

	foundLogin := false
	foundBuildPush := false
	for _, name := range stepNames {
		if name == "Login to Container Registry" {
			foundLogin = true
		}
		if name == "Build and push" {
			foundBuildPush = true
		}
	}
	require.True(t, foundLogin, "workflow should have login step")
	require.True(t, foundBuildPush, "workflow should have build and push step")
}
