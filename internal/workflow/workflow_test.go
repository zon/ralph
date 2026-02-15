package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"gopkg.in/yaml.v3"
)

func TestGenerateWorkflow(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test project file
	projectContent := `name: test-project
description: Test project
requirements:
  - category: test
    description: Test requirement
    items:
      - Test item 1
    passing: false
`
	projectFile := filepath.Join(tmpDir, "project.yaml")
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Create .ralph directory
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	// Create config.yaml
	configContent := `workflow:
  image:
    repository: my-registry/ralph
    tag: v1.0.0
  configMaps:
    - my-config
  secrets:
    - my-secret
  env:
    MY_VAR: my-value
  context: my-context
  namespace: my-namespace
`
	configFile := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create instructions.md
	instructionsContent := "# Custom Instructions\n\nTest instructions"
	instructionsFile := filepath.Join(ralphDir, "instructions.md")
	if err := os.WriteFile(instructionsFile, []byte(instructionsContent), 0644); err != nil {
		t.Fatalf("Failed to create instructions file: %v", err)
	}

	// Change to temp directory for the test
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize a git repository in the temp directory
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Create execution context
	ctx := &execcontext.Context{
		ProjectFile: projectFile,
	}

	// Generate workflow using the testable function
	repoURL := "git@github.com:test/repo.git"
	branch := "main"
	workflowYAML, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, branch)
	if err != nil {
		t.Fatalf("GenerateWorkflowWithGitInfo failed: %v", err)
	}

	// Parse the generated YAML
	var workflow map[string]interface{}
	if err := yaml.Unmarshal([]byte(workflowYAML), &workflow); err != nil {
		t.Fatalf("Failed to parse generated workflow YAML: %v", err)
	}

	// Verify basic structure
	if workflow["apiVersion"] != "argoproj.io/v1alpha1" {
		t.Errorf("apiVersion = %v, want argoproj.io/v1alpha1", workflow["apiVersion"])
	}

	if workflow["kind"] != "Workflow" {
		t.Errorf("kind = %v, want Workflow", workflow["kind"])
	}

	// Verify metadata
	metadata, ok := workflow["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}

	generateName, ok := metadata["generateName"].(string)
	if !ok || !strings.HasPrefix(generateName, "ralph-test-project-") {
		t.Errorf("generateName = %v, want prefix ralph-test-project-", generateName)
	}

	// Verify spec exists
	spec, ok := workflow["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("spec is not a map")
	}

	// Verify entrypoint
	if spec["entrypoint"] != "ralph-executor" {
		t.Errorf("entrypoint = %v, want ralph-executor", spec["entrypoint"])
	}

	// Verify TTL strategy
	ttlStrategy, ok := spec["ttlStrategy"].(map[string]interface{})
	if !ok {
		t.Fatal("ttlStrategy is not a map")
	}
	if ttlStrategy["secondsAfterCompletion"] != 86400 {
		t.Errorf("ttlStrategy.secondsAfterCompletion = %v, want 86400", ttlStrategy["secondsAfterCompletion"])
	}

	// Verify podGC
	podGC, ok := spec["podGC"].(map[string]interface{})
	if !ok {
		t.Fatal("podGC is not a map")
	}
	if podGC["strategy"] != "OnWorkflowCompletion" {
		t.Errorf("podGC.strategy = %v, want OnWorkflowCompletion", podGC["strategy"])
	}

	// Verify arguments
	arguments, ok := spec["arguments"].(map[string]interface{})
	if !ok {
		t.Fatal("arguments is not a map")
	}

	parameters, ok := arguments["parameters"].([]interface{})
	if !ok {
		t.Fatal("parameters is not a list")
	}

	// Verify project-file parameter exists
	hasProjectFile := false
	hasConfigYaml := false
	hasInstructionsMd := false
	for _, param := range parameters {
		paramMap, ok := param.(map[string]interface{})
		if !ok {
			continue
		}
		if paramMap["name"] == "project-file" {
			hasProjectFile = true
			if !strings.Contains(paramMap["value"].(string), "test-project") {
				t.Error("project-file parameter does not contain project content")
			}
		}
		if paramMap["name"] == "config-yaml" {
			hasConfigYaml = true
		}
		if paramMap["name"] == "instructions-md" {
			hasInstructionsMd = true
		}
	}

	if !hasProjectFile {
		t.Error("project-file parameter not found")
	}
	if !hasConfigYaml {
		t.Error("config-yaml parameter not found")
	}
	if !hasInstructionsMd {
		t.Error("instructions-md parameter not found")
	}

	// Verify templates
	templates, ok := spec["templates"].([]interface{})
	if !ok {
		t.Fatal("templates is not a list")
	}

	if len(templates) == 0 {
		t.Fatal("templates is empty")
	}

	template, ok := templates[0].(map[string]interface{})
	if !ok {
		t.Fatal("template is not a map")
	}

	// Verify template name
	if template["name"] != "ralph-executor" {
		t.Errorf("template name = %v, want ralph-executor", template["name"])
	}

	// Verify container
	container, ok := template["container"].(map[string]interface{})
	if !ok {
		t.Fatal("container is not a map")
	}

	// Verify image uses configured values
	if container["image"] != "my-registry/ralph:v1.0.0" {
		t.Errorf("container.image = %v, want my-registry/ralph:v1.0.0", container["image"])
	}

	// Verify working directory
	if container["workingDir"] != "/workspace" {
		t.Errorf("container.workingDir = %v, want /workspace", container["workingDir"])
	}

	// Verify environment variables
	env, ok := container["env"].([]interface{})
	if !ok {
		t.Fatal("env is not a list")
	}

	hasGitRepoURL := false
	hasGitBranch := false
	hasProjectFileEnv := false
	hasCustomEnv := false
	for _, envVar := range env {
		envMap, ok := envVar.(map[string]interface{})
		if !ok {
			continue
		}
		if envMap["name"] == "GIT_REPO_URL" {
			hasGitRepoURL = true
		}
		if envMap["name"] == "GIT_BRANCH" {
			hasGitBranch = true
		}
		if envMap["name"] == "PROJECT_FILE" {
			hasProjectFileEnv = true
		}
		if envMap["name"] == "MY_VAR" && envMap["value"] == "my-value" {
			hasCustomEnv = true
		}
	}

	if !hasGitRepoURL {
		t.Error("GIT_REPO_URL environment variable not found")
	}
	if !hasGitBranch {
		t.Error("GIT_BRANCH environment variable not found")
	}
	if !hasProjectFileEnv {
		t.Error("PROJECT_FILE environment variable not found")
	}
	if !hasCustomEnv {
		t.Error("Custom environment variable MY_VAR not found")
	}

	// Verify volume mounts
	volumeMounts, ok := container["volumeMounts"].([]interface{})
	if !ok {
		t.Fatal("volumeMounts is not a list")
	}

	hasGitMount := false
	hasGithubMount := false
	hasOpencodeMount := false
	hasConfigMapMount := false
	hasSecretMount := false
	for _, mount := range volumeMounts {
		mountMap, ok := mount.(map[string]interface{})
		if !ok {
			continue
		}
		if mountMap["name"] == "git-credentials" && mountMap["mountPath"] == "/secrets/git" {
			hasGitMount = true
		}
		if mountMap["name"] == "github-credentials" && mountMap["mountPath"] == "/secrets/github" {
			hasGithubMount = true
		}
		if mountMap["name"] == "opencode-credentials" && mountMap["mountPath"] == "/secrets/opencode" {
			hasOpencodeMount = true
		}
		if mountMap["name"] == "my-config" && mountMap["mountPath"] == "/configmaps/my-config" {
			hasConfigMapMount = true
		}
		if mountMap["name"] == "my-secret" && mountMap["mountPath"] == "/secrets/my-secret" {
			hasSecretMount = true
		}
	}

	if !hasGitMount {
		t.Error("git-credentials volume mount not found")
	}
	if !hasGithubMount {
		t.Error("github-credentials volume mount not found")
	}
	if !hasOpencodeMount {
		t.Error("opencode-credentials volume mount not found")
	}
	if !hasConfigMapMount {
		t.Error("User-specified configMap mount not found")
	}
	if !hasSecretMount {
		t.Error("User-specified secret mount not found")
	}

	// Verify volumes
	volumes, ok := template["volumes"].([]interface{})
	if !ok {
		t.Fatal("volumes is not a list")
	}

	hasGitVolume := false
	hasGithubVolume := false
	hasOpencodeVolume := false
	hasConfigMapVolume := false
	hasSecretVolume := false
	for _, vol := range volumes {
		volMap, ok := vol.(map[string]interface{})
		if !ok {
			continue
		}
		if volMap["name"] == "git-credentials" {
			hasGitVolume = true
		}
		if volMap["name"] == "github-credentials" {
			hasGithubVolume = true
		}
		if volMap["name"] == "opencode-credentials" {
			hasOpencodeVolume = true
		}
		if volMap["name"] == "my-config" {
			hasConfigMapVolume = true
		}
		if volMap["name"] == "my-secret" {
			hasSecretVolume = true
		}
	}

	if !hasGitVolume {
		t.Error("git-credentials volume not found")
	}
	if !hasGithubVolume {
		t.Error("github-credentials volume not found")
	}
	if !hasOpencodeVolume {
		t.Error("opencode-credentials volume not found")
	}
	if !hasConfigMapVolume {
		t.Error("User-specified configMap volume not found")
	}
	if !hasSecretVolume {
		t.Error("User-specified secret volume not found")
	}
}

func TestGenerateWorkflow_DefaultImage(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test project file
	projectContent := `name: test-project
description: Test project
requirements:
  - category: test
    description: Test requirement
    items:
      - Test item 1
    passing: false
`
	projectFile := filepath.Join(tmpDir, "project.yaml")
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Create .ralph directory with minimal config
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	// Create empty config (no workflow section)
	configContent := `maxIterations: 5
`
	configFile := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize a git repository
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Create execution context
	ctx := &execcontext.Context{
		ProjectFile: projectFile,
	}

	// Generate workflow using the testable function
	repoURL := "git@github.com:test/repo.git"
	branch := "main"
	workflowYAML, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, branch)
	if err != nil {
		t.Fatalf("GenerateWorkflowWithGitInfo failed: %v", err)
	}

	// Parse the generated YAML
	var workflow map[string]interface{}
	if err := yaml.Unmarshal([]byte(workflowYAML), &workflow); err != nil {
		t.Fatalf("Failed to parse generated workflow YAML: %v", err)
	}

	// Verify default image is used
	spec := workflow["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	template := templates[0].(map[string]interface{})
	container := template["container"].(map[string]interface{})

	expectedImage := fmt.Sprintf("ghcr.io/zon/ralph:%s", DefaultContainerVersion)
	if container["image"] != expectedImage {
		t.Errorf("container.image = %v, want %v", container["image"], expectedImage)
	}
}

func TestBuildExecutionScript(t *testing.T) {
	script := buildExecutionScript()

	// Verify script contains key elements
	expectedElements := []string{
		"#!/bin/sh",
		"set -e",
		"git clone",
		"GIT_REPO_URL",
		"GIT_BRANCH",
		"mkdir -p ~/.ssh",
		"ssh-privatekey",
		"GITHUB_TOKEN",
		"auth.json",
		"ralph /tmp/project.yaml",
	}

	for _, element := range expectedElements {
		if !strings.Contains(script, element) {
			t.Errorf("Script does not contain expected element: %s", element)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-config", "my-config"},
		{"my_config", "my-config"},
		{"my.config", "my-config"},
		{"MyConfig", "myconfig"},
		{"my_config.map", "my-config-map"},
	}

	for _, tt := range tests {
		result := sanitizeName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSubmitWorkflow_ArgoNotInstalled(t *testing.T) {
	// This test verifies error handling when argo CLI is not installed
	ctx := &execcontext.Context{}
	cfg := &config.RalphConfig{}

	// Save original PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set PATH to empty to simulate argo not being installed
	os.Setenv("PATH", "")

	_, err := SubmitWorkflow(ctx, "workflow: yaml", cfg)
	if err == nil {
		t.Error("Expected error when argo CLI is not installed, got nil")
	}

	if !strings.Contains(err.Error(), "argo CLI not found") {
		t.Errorf("Error message should mention argo CLI not found, got: %v", err)
	}
}
