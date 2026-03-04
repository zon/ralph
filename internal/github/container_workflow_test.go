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
	stepRun := make(map[string]string)
	stepWith := make(map[string]map[string]interface{})
	for _, s := range steps {
		step := s.(map[string]interface{})
		stepName := ""
		if name, ok := step["name"].(string); ok {
			stepName = name
		}
		if uses, ok := step["uses"].(string); ok {
			stepUses[stepName] = uses
		}
		if run, ok := step["run"].(string); ok {
			stepRun[stepName] = run
		}
		if with, ok := step["with"].(map[string]interface{}); ok {
			stepWith[stepName] = with
		}
	}

	foundBuildx := false
	foundLogin := false
	foundBuildPush := false
	for name, uses := range stepUses {
		if uses == "docker/setup-buildx-action@v3" || uses == "docker/setup-buildx-action@v4" {
			foundBuildx = true
		}
		if uses == "docker/login-action@v3" || uses == "docker/login-action@v4" {
			foundLogin = true
		}
		if uses == "docker/build-push-action@v6" || uses == "docker/build-push-action@v7" || uses == "docker/build-push-action@v5" {
			foundBuildPush = true

			with, ok := stepWith[name]
			require.True(t, ok, "build-push-action should have 'with' section")
			require.Contains(t, with, "cache-from", "build-push-action should have cache-from")
			require.Contains(t, with, "cache-to", "build-push-action should have cache-to")

			cacheFrom, ok := with["cache-from"].(string)
			require.True(t, ok, "cache-from should be a string")
			require.Contains(t, cacheFrom, "type=gha", "cache-from should use GitHub Actions cache")

			cacheTo, ok := with["cache-to"].(string)
			require.True(t, ok, "cache-to should be a string")
			require.Contains(t, cacheTo, "type=gha", "cache-to should use GitHub Actions cache")
		}
	}
	require.True(t, foundBuildx, "workflow should use docker/setup-buildx-action")
	require.True(t, foundLogin, "workflow should use docker/login-action")
	require.True(t, foundBuildPush, "workflow should use docker/build-push-action")

	foundPodmanLogin := false
	foundPodmanBuild := false
	foundPodmanPush := false
	for _, run := range stepRun {
		if run != "" {
			if contains(run, "podman login") {
				foundPodmanLogin = true
			}
			if contains(run, "podman build") {
				foundPodmanBuild = true
			}
			if contains(run, "podman push") {
				foundPodmanPush = true
			}
		}
	}
	require.False(t, foundPodmanLogin, "workflow should not use podman login CLI command")
	require.False(t, foundPodmanBuild, "workflow should not use podman build CLI command")
	require.False(t, foundPodmanPush, "workflow should not use podman push CLI command")

	hasLatestTag := false
	hasVersionTag := false
	hasShaTag := false
	for name, uses := range stepUses {
		if uses == "docker/build-push-action@v6" || uses == "docker/build-push-action@v7" || uses == "docker/build-push-action@v5" {
			with, ok := stepWith[name]
			require.True(t, ok, "build-push-action should have 'with' section")
			require.Contains(t, with, "tags", "build-push-action should have tags")

			var tagStrings []string
			if tags, ok := with["tags"].([]interface{}); ok {
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						tagStrings = append(tagStrings, tagStr)
					}
				}
			} else if tagsStr, ok := with["tags"].(string); ok {
				tagStrings = append(tagStrings, tagsStr)
			}

			require.NotEmpty(t, tagStrings, "build-push-action should have tags")
			for _, tagStr := range tagStrings {
				if contains(tagStr, "latest") {
					hasLatestTag = true
				}
				if contains(tagStr, "steps.version.outputs.VERSION") {
					hasVersionTag = true
				}
				if contains(tagStr, "GITHUB_SHA") || contains(tagStr, "SHORT_SHA") {
					hasShaTag = true
				}
			}
		}
	}
	require.True(t, hasLatestTag, "workflow should tag image with latest")
	require.True(t, hasVersionTag, "workflow should tag image with version from VERSION file (3.2.0)")
	require.False(t, hasShaTag, "workflow should NOT tag image with short commit SHA")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
