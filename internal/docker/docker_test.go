package docker

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerfileExists(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	_, err := os.Stat(containerfilePath)
	require.NoError(t, err, "Containerfile should exist at %s", containerfilePath)
}

func TestDockerfileContainsRequiredComponents(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)

	requiredComponents := []struct {
		name        string
		searchTerms []string
	}{
		{
			name:        "Go toolchain",
			searchTerms: []string{"golang", "go"},
		},
		{
			name:        "Bun runtime",
			searchTerms: []string{"bun"},
		},
		{
			name:        "Playwright",
			searchTerms: []string{"playwright"},
		},
		{
			name:        "Ralph binary",
			searchTerms: []string{"ralph"},
		},
		{
			name:        "Git",
			searchTerms: []string{"git"},
		},
		{
			name:        "jq",
			searchTerms: []string{"jq"},
		},
	}

	for _, component := range requiredComponents {
		t.Run(component.name, func(t *testing.T) {
			found := false
			for _, term := range component.searchTerms {
				if strings.Contains(strings.ToLower(dockerfile), strings.ToLower(term)) {
					found = true
					break
				}
			}
			assert.True(t, found, "Containerfile should contain required component: %s", component.name)
		})
	}
}

func TestDockerfileUsesMultiStageBuilds(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)

	assert.True(t, strings.Contains(dockerfile, "AS builder") || strings.Contains(dockerfile, "AS build"),
		"Containerfile should use multi-stage builds")
	assert.Contains(t, dockerfile, "COPY --from=", "Containerfile should copy artifacts from build stage")
}

func TestDockerfileUpgradesNpm(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	containerfilePath := filepath.Join(projectRoot, "Containerfile")

	content, err := os.ReadFile(containerfilePath)
	require.NoError(t, err, "Should be able to read Containerfile")

	dockerfile := string(content)

	assert.Contains(t, dockerfile, "npm install -g npm@${NPM_VERSION}",
		"Containerfile should upgrade the bundled npm to the pinned NPM_VERSION")

	versionPattern := regexp.MustCompile(`ENV\s+NPM_VERSION=(\S+)`)
	match := versionPattern.FindStringSubmatch(dockerfile)
	require.NotNil(t, match, "Containerfile should pin NPM_VERSION")

	major := strings.SplitN(match[1], ".", 2)[0]
	majorVersion, err := strconv.Atoi(major)
	require.NoError(t, err, "NPM_VERSION major should be numeric, got %q", match[1])
	assert.GreaterOrEqual(t, majorVersion, 12,
		"Containerfile should install npm 12 or later, got %s", match[1])
}

func TestPushScriptExists(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	_, err := os.Stat(scriptPath)
	require.NoError(t, err, "Push script should exist at %s", scriptPath)
}

func TestPushScriptIsExecutable(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	info, err := os.Stat(scriptPath)
	require.NoError(t, err, "Should be able to stat push script")

	mode := info.Mode()
	assert.NotZero(t, mode&0111, "Push script should be executable")
}

func TestPushScriptContentsValid(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err, "Should be able to read push script")

	script := string(content)

	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"Shebang", "#!/bin/bash"},
		{"Set error handling", "set -e"},
		{"Repository variable", "REPOSITORY"},
		{"Tag variable", "TAG"},
		{"Podman build command", "podman build"},
		{"Podman push command", "podman push"},
		{"Containerfile reference", "Containerfile"},
	}

	for _, element := range requiredElements {
		t.Run(element.name, func(t *testing.T) {
			assert.Contains(t, script, element.pattern, "Push script should contain: %s", element.name)
		})
	}
}

func TestPushScriptUsesEnvironmentVariables(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err, "Should be able to read push script")

	script := string(content)

	envVarPatterns := []string{
		"RALPH_IMAGE_REPOSITORY",
		"RALPH_IMAGE_TAG",
	}

	for _, pattern := range envVarPatterns {
		t.Run(pattern, func(t *testing.T) {
			assert.Contains(t, script, pattern, "Push script should support %s environment variable", pattern)
		})
	}

	assert.Contains(t, script, "ghcr.io/zon/ralph", "Push script should have default repository")
	assert.Contains(t, script, "latest", "Push script should have default tag")
}

func TestDefaultImageMatchesWorkflowDefault(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-image.sh")

	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err, "Should be able to read push script")

	script := string(content)

	expectedRepo := "ghcr.io/zon/ralph"
	assert.Contains(t, script, expectedRepo, "Script default repository should be %s", expectedRepo)
}
