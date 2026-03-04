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

	stepUses := make(map[string]string)
	stepNames := make([]string, len(steps))
	stepRun := make([]string, len(steps))
	for i, s := range steps {
		step := s.(map[string]interface{})
		stepName := ""
		if name, ok := step["name"].(string); ok {
			stepNames[i] = name
			stepName = name
		}
		if uses, ok := step["uses"].(string); ok {
			stepUses[stepName] = uses
		}
		if run, ok := step["run"].(string); ok {
			stepRun[i] = run
		}
	}

	for uses := range stepUses {
		require.NotContains(t, uses, "docker/", "workflow should not use docker/* actions")
		require.NotContains(t, uses, "docker/setup-buildx-action", "workflow should not use docker/setup-buildx-action")
		require.NotContains(t, uses, "docker/login-action", "workflow should not use docker/login-action")
		require.NotContains(t, uses, "docker/metadata-action", "workflow should not use docker/metadata-action")
		require.NotContains(t, uses, "docker/build-push-action", "workflow should not use docker/build-push-action")
	}

	foundLogin := false
	foundBuild := false
	foundPush := false
	for _, run := range stepRun {
		if run != "" {
			if contains(run, "podman login") {
				foundLogin = true
			}
			if contains(run, "podman build") {
				foundBuild = true
			}
			if contains(run, "podman push") {
				foundPush = true
			}
		}
	}
	require.True(t, foundLogin, "workflow should use podman login CLI command")
	require.True(t, foundBuild, "workflow should use podman build CLI command")
	require.True(t, foundPush, "workflow should use podman push CLI command")

	hasLatestTag := false
	hasShaTag := false
	for _, run := range stepRun {
		if run != "" {
			if contains(run, "latest") {
				hasLatestTag = true
			}
			if contains(run, "GITHUB_SHA") || contains(run, "SHORT_SHA") {
				hasShaTag = true
			}
		}
	}
	require.True(t, hasLatestTag, "workflow should tag image with latest")
	require.True(t, hasShaTag, "workflow should tag image with short commit SHA")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
