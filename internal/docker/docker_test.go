package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDockerfileExists verifies that the Dockerfile exists in the project root
func TestDockerfileExists(t *testing.T) {
	// Get project root (go up from internal/docker to root)
	projectRoot := filepath.Join("..", "..")
	dockerfilePath := filepath.Join(projectRoot, "Dockerfile")

	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		t.Fatalf("Dockerfile does not exist at %s", dockerfilePath)
	}
}

// TestDockerfileContainsRequiredComponents verifies Dockerfile includes all required dependencies
func TestDockerfileContainsRequiredComponents(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	dockerfilePath := filepath.Join(projectRoot, "Dockerfile")

	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("Failed to read Dockerfile: %v", err)
	}

	dockerfile := string(content)

	// Required components
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
	}

	for _, component := range requiredComponents {
		found := false
		for _, term := range component.searchTerms {
			if strings.Contains(strings.ToLower(dockerfile), strings.ToLower(term)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Dockerfile is missing required component: %s", component.name)
		}
	}
}

// TestDockerfileUsesMultiStageBuilds verifies Dockerfile uses multi-stage builds
func TestDockerfileUsesMultiStageBuilds(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	dockerfilePath := filepath.Join(projectRoot, "Dockerfile")

	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("Failed to read Dockerfile: %v", err)
	}

	dockerfile := string(content)

	// Check for multi-stage build pattern (FROM ... AS ...)
	if !strings.Contains(dockerfile, "AS builder") && !strings.Contains(dockerfile, "AS build") {
		t.Error("Dockerfile should use multi-stage builds for efficient image size")
	}

	// Check for COPY --from pattern
	if !strings.Contains(dockerfile, "COPY --from=") {
		t.Error("Dockerfile should copy artifacts from build stage")
	}
}

// TestPushScriptExists verifies that the push script exists
func TestPushScriptExists(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-default-image.sh")

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("Push script does not exist at %s", scriptPath)
	}
}

// TestPushScriptIsExecutable verifies that the push script has execute permissions
func TestPushScriptIsExecutable(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-default-image.sh")

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("Failed to stat push script: %v", err)
	}

	mode := info.Mode()
	if mode&0111 == 0 {
		t.Error("Push script is not executable (missing execute permission)")
	}
}

// TestPushScriptContentsValid verifies the push script contains required components
func TestPushScriptContentsValid(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-default-image.sh")

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read push script: %v", err)
	}

	script := string(content)

	// Required script components
	requiredElements := []struct {
		name    string
		pattern string
	}{
		{"Shebang", "#!/bin/bash"},
		{"Set error handling", "set -e"},
		{"Repository variable", "REPOSITORY"},
		{"Tag variable", "TAG"},
		{"Docker build command", "docker build"},
		{"Docker push command", "docker push"},
		{"Dockerfile reference", "Dockerfile"},
	}

	for _, element := range requiredElements {
		if !strings.Contains(script, element.pattern) {
			t.Errorf("Push script is missing required element: %s (pattern: %s)", element.name, element.pattern)
		}
	}
}

// TestPushScriptUsesEnvironmentVariables verifies script supports configuration via env vars
func TestPushScriptUsesEnvironmentVariables(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-default-image.sh")

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read push script: %v", err)
	}

	script := string(content)

	// Check for environment variable usage with defaults
	envVarPatterns := []string{
		"RALPH_IMAGE_REPOSITORY",
		"RALPH_IMAGE_TAG",
	}

	for _, pattern := range envVarPatterns {
		if !strings.Contains(script, pattern) {
			t.Errorf("Push script should support %s environment variable", pattern)
		}
	}

	// Check for default values
	if !strings.Contains(script, "ghcr.io/zon/ralph") {
		t.Error("Push script should have default repository")
	}

	if !strings.Contains(script, "latest") {
		t.Error("Push script should have default tag")
	}
}

// TestDefaultImageMatchesWorkflowDefault verifies the default image matches workflow default
func TestDefaultImageMatchesWorkflowDefault(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	scriptPath := filepath.Join(projectRoot, "scripts", "push-default-image.sh")

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read push script: %v", err)
	}

	script := string(content)

	// The default repository should match what's defined in workflow.go
	expectedRepo := "ghcr.io/zon/ralph"
	if !strings.Contains(script, expectedRepo) {
		t.Errorf("Script default repository should be %s to match workflow default", expectedRepo)
	}
}
