package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"gopkg.in/yaml.v3"
)

func TestGenerateWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

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

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

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
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	instructionsContent := "# Custom Instructions\n\nTest instructions"
	if err := os.WriteFile(filepath.Join(ralphDir, "instructions.md"), []byte(instructionsContent), 0644); err != nil {
		t.Fatalf("Failed to create instructions file: %v", err)
	}

	t.Chdir(tmpDir)
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	ctx := &execcontext.Context{}
	ctx.SetProjectFile(projectFile)
	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	projectBranch := "test-project"
	relProjectPath := "project.yaml"

	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, cloneBranch, projectBranch, relProjectPath, false)
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")
	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var workflow map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &workflow), "Failed to parse generated workflow YAML")

	assert.Equal(t, "argoproj.io/v1alpha1", workflow["apiVersion"])
	assert.Equal(t, "Workflow", workflow["kind"])

	metadata, ok := workflow["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata is not a map")
	generateName, ok := metadata["generateName"].(string)
	require.True(t, ok && strings.HasPrefix(generateName, "ralph-test-project-"), "generateName = %v, want prefix ralph-test-project-", generateName)

	spec, ok := workflow["spec"].(map[string]interface{})
	require.True(t, ok, "spec is not a map")
	assert.Equal(t, "ralph-executor", spec["entrypoint"])

	ttlStrategy, ok := spec["ttlStrategy"].(map[string]interface{})
	require.True(t, ok, "ttlStrategy is not a map")
	assert.Equal(t, 86400, ttlStrategy["secondsAfterCompletion"])

	podGC, ok := spec["podGC"].(map[string]interface{})
	require.True(t, ok, "podGC is not a map")
	assert.Equal(t, "OnWorkflowCompletion", podGC["strategy"])
	assert.Equal(t, "10m", podGC["deleteDelayDuration"])

	arguments, ok := spec["arguments"].(map[string]interface{})
	require.True(t, ok, "arguments is not a map")
	parameters, ok := arguments["parameters"].([]interface{})
	require.True(t, ok, "parameters is not a list")

	hasProjectPath, hasInstructionsMd := false, false
	for _, param := range parameters {
		paramMap, ok := param.(map[string]interface{})
		if !ok {
			continue
		}
		if paramMap["name"] == "project-path" {
			hasProjectPath = true
			assert.Equal(t, "project.yaml", paramMap["value"])
		}
		if paramMap["name"] == "instructions-md" {
			hasInstructionsMd = true
		}
	}
	assert.True(t, hasProjectPath, "project-path parameter not found")
	assert.True(t, hasInstructionsMd, "instructions-md parameter not found")

	templates, ok := spec["templates"].([]interface{})
	require.True(t, ok && len(templates) > 0, "templates is empty or not a list")
	tmpl, ok := templates[0].(map[string]interface{})
	require.True(t, ok, "template is not a map")
	assert.Equal(t, "ralph-executor", tmpl["name"])

	container, ok := tmpl["container"].(map[string]interface{})
	require.True(t, ok, "container is not a map")
	assert.Equal(t, "my-registry/ralph:v1.0.0", container["image"])
	assert.Equal(t, "/workspace", container["workingDir"])

	env, ok := container["env"].([]interface{})
	require.True(t, ok, "env is not a list")

	hasGitRepoURL, hasGitBranch, hasProjectBranch, hasCustomEnv, hasBaseBranch, hasPulumiToken := false, false, false, false, false, false
	for _, envVar := range env {
		envMap, ok := envVar.(map[string]interface{})
		if !ok {
			continue
		}
		switch envMap["name"] {
		case "GIT_REPO_URL":
			hasGitRepoURL = true
		case "GIT_BRANCH":
			hasGitBranch = true
			assert.Equal(t, cloneBranch, envMap["value"])
		case "PROJECT_BRANCH":
			hasProjectBranch = true
			assert.Equal(t, projectBranch, envMap["value"])
		case "MY_VAR":
			if envMap["value"] == "my-value" {
				hasCustomEnv = true
			}
		case "BASE_BRANCH":
			hasBaseBranch = true
		case "PULUMI_ACCESS_TOKEN":
			hasPulumiToken = true
			valueFrom, ok := envMap["valueFrom"].(map[string]interface{})
			require.True(t, ok, "PULUMI_ACCESS_TOKEN should have valueFrom")
			secretKeyRef, ok := valueFrom["secretKeyRef"].(map[string]interface{})
			require.True(t, ok, "PULUMI_ACCESS_TOKEN should have secretKeyRef")
			assert.Equal(t, "pulumi-credentials", secretKeyRef["name"])
			assert.Equal(t, "PULUMI_ACCESS_TOKEN", secretKeyRef["key"])
			assert.Equal(t, true, secretKeyRef["optional"])
		}
	}
	assert.True(t, hasGitRepoURL, "GIT_REPO_URL environment variable not found")
	assert.True(t, hasGitBranch, "GIT_BRANCH environment variable not found")
	assert.True(t, hasProjectBranch, "PROJECT_BRANCH environment variable not found")
	assert.True(t, hasCustomEnv, "Custom environment variable MY_VAR not found")
	assert.True(t, hasBaseBranch, "BASE_BRANCH environment variable not found")
	assert.True(t, hasPulumiToken, "PULUMI_ACCESS_TOKEN environment variable not found")

	volumeMounts, ok := container["volumeMounts"].([]interface{})
	require.True(t, ok, "volumeMounts is not a list")

	hasGithubMount, hasOpencodeMount, hasPulumiMount, hasConfigMapMount, hasSecretMount := false, false, false, false, false
	for _, mount := range volumeMounts {
		mountMap, ok := mount.(map[string]interface{})
		if !ok {
			continue
		}
		switch {
		case mountMap["name"] == "github-credentials" && mountMap["mountPath"] == "/secrets/github":
			hasGithubMount = true
		case mountMap["name"] == "opencode-credentials" && mountMap["mountPath"] == "/secrets/opencode":
			hasOpencodeMount = true
		case mountMap["name"] == "pulumi-credentials" && mountMap["mountPath"] == "/secrets/pulumi":
			hasPulumiMount = true
		case mountMap["name"] == "my-config" && mountMap["mountPath"] == "/configmaps/my-config":
			hasConfigMapMount = true
		case mountMap["name"] == "my-secret" && mountMap["mountPath"] == "/secrets/my-secret":
			hasSecretMount = true
		}
	}
	assert.True(t, hasGithubMount, "github-credentials volume mount not found")
	assert.True(t, hasOpencodeMount, "opencode-credentials volume mount not found")
	assert.True(t, hasPulumiMount, "pulumi-credentials volume mount not found")
	assert.True(t, hasConfigMapMount, "User-specified configMap mount not found")
	assert.True(t, hasSecretMount, "User-specified secret mount not found")

	volumes, ok := tmpl["volumes"].([]interface{})
	require.True(t, ok, "volumes is not a list")

	hasGithubVolume, hasOpencodeVolume, hasPulumiVolume, hasConfigMapVolume, hasSecretVolume := false, false, false, false, false
	for _, vol := range volumes {
		volMap, ok := vol.(map[string]interface{})
		if !ok {
			continue
		}
		switch volMap["name"] {
		case "github-credentials":
			hasGithubVolume = true
		case "opencode-credentials":
			hasOpencodeVolume = true
		case "pulumi-credentials":
			hasPulumiVolume = true
			secret, ok := volMap["secret"].(map[string]interface{})
			require.True(t, ok, "pulumi-credentials volume should have secret map")
			assert.Equal(t, true, secret["optional"])
		case "my-config":
			hasConfigMapVolume = true
		case "my-secret":
			hasSecretVolume = true
		}
	}
	assert.True(t, hasGithubVolume, "github-credentials volume not found")
	assert.True(t, hasOpencodeVolume, "opencode-credentials volume not found")
	assert.True(t, hasPulumiVolume, "pulumi-credentials volume not found")
	assert.True(t, hasConfigMapVolume, "User-specified configMap volume not found")
	assert.True(t, hasSecretVolume, "User-specified secret volume not found")

	synchronization, ok := spec["synchronization"].(map[string]interface{})
	require.True(t, ok, "synchronization is not a map")
	mutexes, ok := synchronization["mutexes"].([]interface{})
	require.True(t, ok, "mutexes is not a slice")
	require.NotEmpty(t, mutexes, "mutexes slice is empty")
	mutex, ok := mutexes[0].(map[string]interface{})
	require.True(t, ok, "mutex is not a map")
	mutexName, ok := mutex["name"].(string)
	require.True(t, ok, "mutex name is not a string")
	expectedMutexName := "test-project"
	assert.Equal(t, expectedMutexName, mutexName)
}

func TestGenerateWorkflow_DefaultImage(t *testing.T) {
	tmpDir := t.TempDir()

	projectContent := `name: test-project
description: Test project
requirements:
  - category: test
    description: Test requirement
    items:
      - Test item 1
    passing: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "project.yaml"), []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("maxIterations: 5\n"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Chdir(tmpDir)
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	ctx := &execcontext.Context{}
	ctx.SetProjectFile(filepath.Join(tmpDir, "project.yaml"))
	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "project.yaml", false)
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")
	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var workflow map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &workflow), "Failed to parse generated workflow YAML")

	spec := workflow["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	expectedImage := fmt.Sprintf("ghcr.io/zon/ralph:%s", DefaultContainerVersion())
	assert.Equal(t, expectedImage, container["image"])
}

func TestGenerateMergeWorkflow(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(tmpDir, "project.yaml"), []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	configContent := `workflow:
  image:
    repository: my-registry/ralph
    tag: v2.0.0
`
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Chdir(tmpDir)

	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	prBranch := "ralph/test-project"

	mw, err := GenerateMergeWorkflowWithGitInfo(repoURL, cloneBranch, prBranch, "")
	require.NoError(t, err, "GenerateMergeWorkflowWithGitInfo failed")
	workflowYAML, err := mw.Render()
	require.NoError(t, err, "Render failed")

	var wf map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wf), "Failed to parse generated workflow YAML")

	assert.Equal(t, "argoproj.io/v1alpha1", wf["apiVersion"])
	assert.Equal(t, "Workflow", wf["kind"])

	metadata, ok := wf["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata is not a map")
	assert.Equal(t, "ralph-merge-", metadata["generateName"])

	spec, ok := wf["spec"].(map[string]interface{})
	require.True(t, ok, "spec is not a map")
	assert.Equal(t, "ralph-merger", spec["entrypoint"])

	ttlStrategy, ok := spec["ttlStrategy"].(map[string]interface{})
	require.True(t, ok, "ttlStrategy is not a map")
	assert.Equal(t, 86400, ttlStrategy["secondsAfterCompletion"])

	podGC, ok := spec["podGC"].(map[string]interface{})
	require.True(t, ok, "podGC is not a map")
	assert.Equal(t, "OnWorkflowCompletion", podGC["strategy"])

	templates, ok := spec["templates"].([]interface{})
	require.True(t, ok && len(templates) > 0, "templates is empty or not a list")
	tmpl, ok := templates[0].(map[string]interface{})
	require.True(t, ok, "template is not a map")
	assert.Equal(t, "ralph-merger", tmpl["name"])

	container, ok := tmpl["container"].(map[string]interface{})
	require.True(t, ok, "container is not a map")
	assert.Equal(t, "my-registry/ralph:v2.0.0", container["image"])

	env, ok := container["env"].([]interface{})
	require.True(t, ok, "env is not a list")

	hasPRBranch, hasGitRepoURL, hasGitBranch := false, false, false
	for _, e := range env {
		em, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		switch em["name"] {
		case "PR_BRANCH":
			hasPRBranch = true
			assert.Equal(t, prBranch, em["value"])
		case "GIT_REPO_URL":
			hasGitRepoURL = true
		case "GIT_BRANCH":
			hasGitBranch = true
			assert.Equal(t, cloneBranch, em["value"])
		}
	}
	assert.True(t, hasPRBranch, "PR_BRANCH environment variable not found")
	assert.True(t, hasGitRepoURL, "GIT_REPO_URL environment variable not found")
	assert.True(t, hasGitBranch, "GIT_BRANCH environment variable not found")

	volumes, ok := tmpl["volumes"].([]interface{})
	require.True(t, ok, "volumes is not a list")
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
	assert.True(t, hasGithubVol, "github-credentials volume not found")

	synchronization, ok := spec["synchronization"].(map[string]interface{})
	require.True(t, ok, "synchronization is not a map")
	mutexes, ok := synchronization["mutexes"].([]interface{})
	require.True(t, ok, "mutexes is not a slice")
	require.NotEmpty(t, mutexes, "mutexes slice is empty")
	mutex, ok := mutexes[0].(map[string]interface{})
	require.True(t, ok, "mutex is not a map")
	mutexName, ok := mutex["name"].(string)
	require.True(t, ok, "mutex name is not a string")
	expectedMutexName := "ralph-test-project"
	assert.Equal(t, expectedMutexName, mutexName)
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
	if err := os.WriteFile(filepath.Join(tmpDir, "project.yaml"), []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("maxIterations: 5\n"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Chdir(tmpDir)

	mw, err := GenerateMergeWorkflowWithGitInfo("git@github.com:test/repo.git", "main", "ralph/test", "")
	require.NoError(t, err, "GenerateMergeWorkflowWithGitInfo failed")
	workflowYAML, err := mw.Render()
	require.NoError(t, err, "Render failed")

	var wf map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wf), "Failed to parse generated workflow YAML")

	spec := wf["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	expectedImage := fmt.Sprintf("ghcr.io/zon/ralph:%s", DefaultContainerVersion())
	assert.Equal(t, expectedImage, container["image"])
}

func TestSubmitWorkflow_ArgoNotInstalled(t *testing.T) {
	wf := &Workflow{RalphConfig: &config.RalphConfig{}}

	t.Setenv("PATH", "")

	_, err := wf.Submit("test-namespace")
	require.Error(t, err, "Expected error when argo CLI is not installed")
	assert.True(t, strings.Contains(err.Error(), "argo CLI not found"), "Error message should mention argo CLI not found, got: %v", err)
}

func TestExtractWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "parses workflow name from Name field",
			output:   "Name: ralph-test-abc123\nNamespace: default\nStatus: Succeeded",
			expected: "ralph-test-abc123",
		},
		{
			name:     "returns empty string when Name field not present",
			output:   "Namespace: default\nStatus: Succeeded\nWorkflow submitted successfully",
			expected: "",
		},
		{
			name:     "handles multi-line output and extracts from correct line",
			output:   "Workflow submitted successfully\nName: ralph-feature-xyz789\nNamespace: default\nStatus: Running",
			expected: "ralph-feature-xyz789",
		},
		{
			name:     "handles Name field with extra whitespace",
			output:   "Name:    ralph-test-spaces\nStatus: Succeeded",
			expected: "ralph-test-spaces",
		},
		{
			name:     "returns empty string for empty output",
			output:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWorkflowName(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkflowRender_CommentScriptBranching(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("maxIterations: 5\n"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	commentBody := "Please review this PR"
	prNumber := "123"

	wf := &Workflow{
		ProjectName:   "test-project",
		RepoURL:       "https://github.com/owner/repo.git",
		RepoOwner:     "owner",
		RepoName:      "repo",
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		CommentBody:   commentBody,
		PRNumber:      prNumber,
		RalphConfig:   &config.RalphConfig{},
	}

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	assert.True(t, strings.Contains(workflowYAML, commentBody), "Rendered YAML should contain comment body %q", commentBody)
	assert.True(t, strings.Contains(workflowYAML, prNumber), "Rendered YAML should contain PR number %q", prNumber)

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	args := container["args"].([]interface{})
	script := args[0].(string)

	assert.True(t, strings.Contains(script, "ralph comment"), "Script should contain 'ralph comment' for comment-triggered workflow")
}

func TestWorkflowRender_RunScriptBranching(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("maxIterations: 5\n"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	wf := &Workflow{
		ProjectName:   "test-project",
		RepoURL:       "https://github.com/owner/repo.git",
		RepoOwner:     "owner",
		RepoName:      "repo",
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		CommentBody:   "",
		PRNumber:      "",
		RalphConfig:   &config.RalphConfig{},
	}

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	args := container["args"].([]interface{})
	script := args[0].(string)

	assert.True(t, strings.Contains(script, "ralph run") || strings.Contains(script, "ralph_run"), "Script should contain 'ralph run' for regular workflow")
	assert.False(t, strings.Contains(script, "ralph comment"), "Script should NOT contain 'ralph comment' when CommentBody is empty")
}

func TestMergeWorkflowRender_EnvVarCoverage(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("maxIterations: 5\n"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	repoOwner := "test-owner"
	repoName := "test-repo"
	prNumber := "456"

	mw := &MergeWorkflow{
		RepoURL:     "https://github.com/test-owner/test-repo.git",
		RepoOwner:   repoOwner,
		RepoName:    repoName,
		CloneBranch: "main",
		PRBranch:    "feature-branch",
		PRNumber:    prNumber,
		RalphConfig: &config.RalphConfig{},
	}

	workflowYAML, err := mw.Render()
	require.NoError(t, err, "Render failed")

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	env := container["env"].([]interface{})

	hasRepoOwner, hasRepoName, hasPRNumber := false, false, false
	for _, e := range env {
		em := e.(map[string]interface{})
		switch em["name"] {
		case "GITHUB_REPO_OWNER":
			hasRepoOwner = true
			assert.Equal(t, repoOwner, em["value"])
		case "GITHUB_REPO_NAME":
			hasRepoName = true
			assert.Equal(t, repoName, em["value"])
		case "PR_NUMBER":
			hasPRNumber = true
			assert.Equal(t, prNumber, em["value"])
		}
	}
	assert.True(t, hasRepoOwner, "GITHUB_REPO_OWNER environment variable not found")
	assert.True(t, hasRepoName, "GITHUB_REPO_NAME environment variable not found")
	assert.True(t, hasPRNumber, "PR_NUMBER environment variable not found")
}

func TestMergeWorkflowRender_GitHubCredentialsVolumeMount(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("maxIterations: 5\n"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	mw := &MergeWorkflow{
		RepoURL:     "https://github.com/test-owner/test-repo.git",
		RepoOwner:   "test-owner",
		RepoName:    "test-repo",
		CloneBranch: "main",
		PRBranch:    "feature-branch",
		RalphConfig: &config.RalphConfig{},
	}

	workflowYAML, err := mw.Render()
	require.NoError(t, err, "Render failed")

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	volumeMounts := container["volumeMounts"].([]interface{})

	hasGithubMount := false
	for _, m := range volumeMounts {
		mount := m.(map[string]interface{})
		if mount["name"] == "github-credentials" && mount["mountPath"] == "/secrets/github" {
			hasGithubMount = true
			assert.Equal(t, true, mount["readOnly"], "github-credentials volume mount should have readOnly set to true")
		}
	}
	assert.True(t, hasGithubMount, "github-credentials volume mount at /secrets/github not found")
}

func TestBaseBranchOverride(t *testing.T) {
	tmpDir := t.TempDir()

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

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	configContent := `defaultBranch: config-base-branch
workflow:
  namespace: my-namespace
`
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Chdir(tmpDir)
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	ctx := &execcontext.Context{}
	ctx.SetProjectFile(projectFile)
	ctx.SetBaseBranch("override-branch")

	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	projectBranch := "test-project"
	relProjectPath := "project.yaml"

	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, cloneBranch, projectBranch, relProjectPath, false)
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")

	assert.Equal(t, "override-branch", wf.BaseBranch, "BaseBranch should be set from context")

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var workflow map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &workflow), "Failed to parse generated workflow YAML")

	spec := workflow["spec"].(map[string]interface{})

	arguments := spec["arguments"].(map[string]interface{})
	params := arguments["parameters"].([]interface{})

	var baseBranchParamValue string
	var hasBaseBranchParam bool
	for _, p := range params {
		paramMap := p.(map[string]interface{})
		if paramMap["name"] == "base-branch" {
			hasBaseBranchParam = true
			baseBranchParamValue = paramMap["value"].(string)
			break
		}
	}
	assert.True(t, hasBaseBranchParam, "base-branch parameter should exist")
	assert.Equal(t, "override-branch", baseBranchParamValue, "base-branch parameter should be override-branch")

	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	env := container["env"].([]interface{})

	var baseBranchValue string
	for _, envVar := range env {
		envMap := envVar.(map[string]interface{})
		if envMap["name"] == "BASE_BRANCH" {
			baseBranchValue = envMap["value"].(string)
			break
		}
	}

	assert.Equal(t, "{{workflow.parameters.base-branch}}", baseBranchValue, "BASE_BRANCH env var should reference workflow parameter")
}

func TestBaseBranchDefault(t *testing.T) {
	tmpDir := t.TempDir()

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

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	configContent := `defaultBranch: main
workflow:
  namespace: my-namespace
`
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Chdir(tmpDir)
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	ctx := &execcontext.Context{}
	ctx.SetProjectFile(projectFile)

	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	projectBranch := "test-project"
	relProjectPath := "project.yaml"

	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, cloneBranch, projectBranch, relProjectPath, false)
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")

	assert.Equal(t, "", wf.BaseBranch, "BaseBranch should be empty when not set")

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var workflow map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &workflow), "Failed to parse generated workflow YAML")

	spec := workflow["spec"].(map[string]interface{})

	arguments := spec["arguments"].(map[string]interface{})
	params := arguments["parameters"].([]interface{})

	var baseBranchParamValue string
	var hasBaseBranchParam bool
	for _, p := range params {
		paramMap := p.(map[string]interface{})
		if paramMap["name"] == "base-branch" {
			hasBaseBranchParam = true
			baseBranchParamValue = paramMap["value"].(string)
			break
		}
	}
	assert.True(t, hasBaseBranchParam, "base-branch parameter should exist")
	assert.Equal(t, "main", baseBranchParamValue, "base-branch parameter should default to config defaultBranch")
}
