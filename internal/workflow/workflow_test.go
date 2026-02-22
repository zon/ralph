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
    - name: my-config
  secrets:
    - name: my-secret
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
	cloneBranch := "main"
	projectBranch := "test-project"
	relProjectPath := "project.yaml"
	dryRun := false
	verbose := false
	workflowYAML, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, cloneBranch, projectBranch, relProjectPath, dryRun, verbose)
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
	if podGC["deleteDelayDuration"] != "10m" {
		t.Errorf("podGC.deleteDelayDuration = %v, want 10m", podGC["deleteDelayDuration"])
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

	// Verify parameters exist
	hasProjectPath := false
	hasConfigYaml := false
	hasInstructionsMd := false
	for _, param := range parameters {
		paramMap, ok := param.(map[string]interface{})
		if !ok {
			continue
		}
		if paramMap["name"] == "project-path" {
			hasProjectPath = true
			if paramMap["value"] != "project.yaml" {
				t.Errorf("project-path = %v, want project.yaml", paramMap["value"])
			}
		}
		if paramMap["name"] == "config-yaml" {
			hasConfigYaml = true
		}
		if paramMap["name"] == "instructions-md" {
			hasInstructionsMd = true
		}
	}

	if !hasProjectPath {
		t.Error("project-path parameter not found")
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
	hasProjectBranch := false
	hasCustomEnv := false
	hasBaseBranch := false
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
			if envMap["value"] != cloneBranch {
				t.Errorf("GIT_BRANCH = %v, want %v", envMap["value"], cloneBranch)
			}
		}
		if envMap["name"] == "PROJECT_BRANCH" {
			hasProjectBranch = true
			if envMap["value"] != projectBranch {
				t.Errorf("PROJECT_BRANCH = %v, want %v", envMap["value"], projectBranch)
			}
		}
		if envMap["name"] == "MY_VAR" && envMap["value"] == "my-value" {
			hasCustomEnv = true
		}
		if envMap["name"] == "BASE_BRANCH" {
			hasBaseBranch = true
		}
	}

	if !hasGitRepoURL {
		t.Error("GIT_REPO_URL environment variable not found")
	}
	if !hasGitBranch {
		t.Error("GIT_BRANCH environment variable not found")
	}
	if !hasProjectBranch {
		t.Error("PROJECT_BRANCH environment variable not found")
	}
	if !hasCustomEnv {
		t.Error("Custom environment variable MY_VAR not found")
	}
	if !hasBaseBranch {
		t.Error("BASE_BRANCH environment variable not found")
	}

	// Verify volume mounts
	volumeMounts, ok := container["volumeMounts"].([]interface{})
	if !ok {
		t.Fatal("volumeMounts is not a list")
	}

	hasGithubMount := false
	hasOpencodeMount := false
	hasConfigMapMount := false
	hasSecretMount := false
	for _, mount := range volumeMounts {
		mountMap, ok := mount.(map[string]interface{})
		if !ok {
			continue
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

	hasGithubVolume := false
	hasOpencodeVolume := false
	hasConfigMapVolume := false
	hasSecretVolume := false
	for _, vol := range volumes {
		volMap, ok := vol.(map[string]interface{})
		if !ok {
			continue
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
	cloneBranch := "main"
	projectBranch := "test-project"
	relProjectPath := "project.yaml"
	dryRun := false
	verbose := false
	workflowYAML, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, cloneBranch, projectBranch, relProjectPath, dryRun, verbose)
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

	expectedImage := fmt.Sprintf("ghcr.io/zon/ralph:%s", DefaultContainerVersion())
	if container["image"] != expectedImage {
		t.Errorf("container.image = %v, want %v", container["image"], expectedImage)
	}
}

func TestBuildExecutionScript(t *testing.T) {
	tests := []struct {
		name            string
		dryRun          bool
		verbose         bool
		expectedCommand string
	}{
		{
			name:            "no flags",
			dryRun:          false,
			verbose:         false,
			expectedCommand: "ralph \"$PROJECT_PATH\" --local --no-notify",
		},
		{
			name:            "dry-run only",
			dryRun:          true,
			verbose:         false,
			expectedCommand: "ralph \"$PROJECT_PATH\" --local --dry-run --no-notify",
		},
		{
			name:            "verbose only",
			dryRun:          false,
			verbose:         true,
			expectedCommand: "ralph \"$PROJECT_PATH\" --local --verbose --no-notify",
		},
		{
			name:            "both flags",
			dryRun:          true,
			verbose:         true,
			expectedCommand: "ralph \"$PROJECT_PATH\" --local --dry-run --verbose --no-notify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal config for testing
			cfg := &config.RalphConfig{
				Workflow: config.WorkflowConfig{},
			}
			script := buildExecutionScript(tt.dryRun, tt.verbose, cfg)

			// Verify script contains key elements
			expectedElements := []string{
				"#!/bin/sh",
				"set -e",
				"git clone",
				"GIT_REPO_URL",
				"GIT_BRANCH",
				"PROJECT_BRANCH",
				"BASE_BRANCH",
				"ralph github-token",
				"x-access-token:${GITHUB_TOKEN}@github.com",
				config.DefaultAppName + "[bot]",
				config.DefaultAppName + "[bot]@users.noreply.github.com",
				"auth.json",
				tt.expectedCommand,
			}

			for _, element := range expectedElements {
				if !strings.Contains(script, element) {
					t.Errorf("Script does not contain expected element: %s", element)
				}
			}
		})
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

func TestGenerateMergeWorkflow(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test project file with all requirements passing
	projectContent := `name: test-project
description: Test project
requirements:
  - category: test
    description: Test requirement
    items:
      - Test item 1
    passing: true
`
	projectFile := filepath.Join(tmpDir, "project.yaml")
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Create .ralph directory with config
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	configContent := `workflow:
  image:
    repository: my-registry/ralph
    tag: v2.0.0
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

	// Generate merge workflow
	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	prBranch := "ralph/test-project"
	relProjectPath := "project.yaml"

	workflowYAML, err := GenerateMergeWorkflowWithGitInfo(projectFile, repoURL, cloneBranch, prBranch, relProjectPath)
	if err != nil {
		t.Fatalf("GenerateMergeWorkflowWithGitInfo failed: %v", err)
	}

	// Parse the generated YAML
	var wf map[string]interface{}
	if err := yaml.Unmarshal([]byte(workflowYAML), &wf); err != nil {
		t.Fatalf("Failed to parse generated workflow YAML: %v", err)
	}

	// Verify basic structure
	if wf["apiVersion"] != "argoproj.io/v1alpha1" {
		t.Errorf("apiVersion = %v, want argoproj.io/v1alpha1", wf["apiVersion"])
	}
	if wf["kind"] != "Workflow" {
		t.Errorf("kind = %v, want Workflow", wf["kind"])
	}

	// Verify metadata
	metadata, ok := wf["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}
	if metadata["generateName"] != "ralph-merge-" {
		t.Errorf("generateName = %v, want ralph-merge-", metadata["generateName"])
	}

	// Verify spec
	spec, ok := wf["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("spec is not a map")
	}
	if spec["entrypoint"] != "ralph-merger" {
		t.Errorf("entrypoint = %v, want ralph-merger", spec["entrypoint"])
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

	// Verify arguments / parameters
	arguments, ok := spec["arguments"].(map[string]interface{})
	if !ok {
		t.Fatal("arguments is not a map")
	}
	params, ok := arguments["parameters"].([]interface{})
	if !ok {
		t.Fatal("parameters is not a list")
	}

	hasProjectPath := false
	for _, p := range params {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if pm["name"] == "project-path" {
			hasProjectPath = true
			if pm["value"] != "project.yaml" {
				t.Errorf("project-path = %v, want project.yaml", pm["value"])
			}
		}
	}
	if !hasProjectPath {
		t.Error("project-path parameter not found")
	}

	// Verify template
	templates, ok := spec["templates"].([]interface{})
	if !ok || len(templates) == 0 {
		t.Fatal("templates is empty or not a list")
	}
	tmpl, ok := templates[0].(map[string]interface{})
	if !ok {
		t.Fatal("template is not a map")
	}
	if tmpl["name"] != "ralph-merger" {
		t.Errorf("template name = %v, want ralph-merger", tmpl["name"])
	}

	// Verify container
	container, ok := tmpl["container"].(map[string]interface{})
	if !ok {
		t.Fatal("container is not a map")
	}
	if container["image"] != "my-registry/ralph:v2.0.0" {
		t.Errorf("container.image = %v, want my-registry/ralph:v2.0.0", container["image"])
	}

	// Verify environment variables contain PR_BRANCH
	env, ok := container["env"].([]interface{})
	if !ok {
		t.Fatal("env is not a list")
	}

	hasPRBranch := false
	hasGitRepoURL := false
	hasGitBranch := false
	for _, e := range env {
		em, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		if em["name"] == "PR_BRANCH" {
			hasPRBranch = true
			if em["value"] != prBranch {
				t.Errorf("PR_BRANCH = %v, want %v", em["value"], prBranch)
			}
		}
		if em["name"] == "GIT_REPO_URL" {
			hasGitRepoURL = true
		}
		if em["name"] == "GIT_BRANCH" {
			hasGitBranch = true
			if em["value"] != cloneBranch {
				t.Errorf("GIT_BRANCH = %v, want %v", em["value"], cloneBranch)
			}
		}
	}
	if !hasPRBranch {
		t.Error("PR_BRANCH environment variable not found")
	}
	if !hasGitRepoURL {
		t.Error("GIT_REPO_URL environment variable not found")
	}
	if !hasGitBranch {
		t.Error("GIT_BRANCH environment variable not found")
	}

	// Verify volumes contain git-credentials and github-credentials
	volumes, ok := tmpl["volumes"].([]interface{})
	if !ok {
		t.Fatal("volumes is not a list")
	}
	hasGithubVol := false
	for _, v := range volumes {
		vm, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if vm["name"] == "github-credentials" {
			hasGithubVol = true
		}
	}
	if !hasGithubVol {
		t.Error("github-credentials volume not found")
	}
}

func TestBuildMergeScript(t *testing.T) {
	tests := []struct {
		name             string
		expectStrings    []string
		notExpectStrings []string
	}{
		{
			name: "merge script",
			expectStrings: []string{
				"#!/bin/sh",
				"set -e",
				"git clone",
				"GIT_REPO_URL",
				"GIT_BRANCH",
				"PR_BRANCH",
				"ralph github-token",
				"x-access-token:${GITHUB_TOKEN}@github.com",
				config.DefaultAppName + "[bot]",
				config.DefaultAppName + "[bot]@users.noreply.github.com",
				"passing: false",
				"rm \"$PROJECT_PATH\"",
				"git add -A",
				"git commit",
				"git push",
				"gh pr merge",
				"--merge",
				"--delete-branch",
			},
			notExpectStrings: []string{
				"mkdir -p ~/.ssh",
				"ssh-privatekey",
				"ssh-keyscan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := buildMergeScript()
			for _, s := range tt.expectStrings {
				if !strings.Contains(script, s) {
					t.Errorf("merge script does not contain expected element: %q", s)
				}
			}
			for _, s := range tt.notExpectStrings {
				if strings.Contains(script, s) {
					t.Errorf("merge script unexpectedly contains: %q", s)
				}
			}
		})
	}
}

func TestGenerateMergeWorkflow_DefaultImage(t *testing.T) {
	tmpDir := t.TempDir()

	projectContent := `name: test-project
description: Test project
requirements:
  - category: test
    description: Test requirement
    items:
      - Test item 1
    passing: true
`
	projectFile := filepath.Join(tmpDir, "project.yaml")
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	// Minimal config with no image settings
	configContent := "maxIterations: 5\n"
	configFile := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	workflowYAML, err := GenerateMergeWorkflowWithGitInfo(projectFile, "git@github.com:test/repo.git", "main", "ralph/test", "project.yaml")
	if err != nil {
		t.Fatalf("GenerateMergeWorkflowWithGitInfo failed: %v", err)
	}

	var wf map[string]interface{}
	if err := yaml.Unmarshal([]byte(workflowYAML), &wf); err != nil {
		t.Fatalf("Failed to parse generated workflow YAML: %v", err)
	}

	spec := wf["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	expectedImage := fmt.Sprintf("ghcr.io/zon/ralph:%s", DefaultContainerVersion())
	if container["image"] != expectedImage {
		t.Errorf("container.image = %v, want %v", container["image"], expectedImage)
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
