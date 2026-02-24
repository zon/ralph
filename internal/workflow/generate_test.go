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

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	ctx := &execcontext.Context{ProjectFile: projectFile}
	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	projectBranch := "test-project"
	relProjectPath := "project.yaml"

	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, cloneBranch, projectBranch, relProjectPath, false, false)
	if err != nil {
		t.Fatalf("GenerateWorkflowWithGitInfo failed: %v", err)
	}
	workflowYAML, err := wf.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var workflow map[string]interface{}
	if err := yaml.Unmarshal([]byte(workflowYAML), &workflow); err != nil {
		t.Fatalf("Failed to parse generated workflow YAML: %v", err)
	}

	if workflow["apiVersion"] != "argoproj.io/v1alpha1" {
		t.Errorf("apiVersion = %v, want argoproj.io/v1alpha1", workflow["apiVersion"])
	}
	if workflow["kind"] != "Workflow" {
		t.Errorf("kind = %v, want Workflow", workflow["kind"])
	}

	metadata, ok := workflow["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}
	generateName, ok := metadata["generateName"].(string)
	if !ok || !strings.HasPrefix(generateName, "ralph-test-project-") {
		t.Errorf("generateName = %v, want prefix ralph-test-project-", generateName)
	}

	spec, ok := workflow["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("spec is not a map")
	}
	if spec["entrypoint"] != "ralph-executor" {
		t.Errorf("entrypoint = %v, want ralph-executor", spec["entrypoint"])
	}

	ttlStrategy, ok := spec["ttlStrategy"].(map[string]interface{})
	if !ok {
		t.Fatal("ttlStrategy is not a map")
	}
	if ttlStrategy["secondsAfterCompletion"] != 86400 {
		t.Errorf("ttlStrategy.secondsAfterCompletion = %v, want 86400", ttlStrategy["secondsAfterCompletion"])
	}

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

	arguments, ok := spec["arguments"].(map[string]interface{})
	if !ok {
		t.Fatal("arguments is not a map")
	}
	parameters, ok := arguments["parameters"].([]interface{})
	if !ok {
		t.Fatal("parameters is not a list")
	}

	hasProjectPath, hasInstructionsMd := false, false
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
		if paramMap["name"] == "instructions-md" {
			hasInstructionsMd = true
		}
	}
	if !hasProjectPath {
		t.Error("project-path parameter not found")
	}
	if !hasInstructionsMd {
		t.Error("instructions-md parameter not found")
	}

	templates, ok := spec["templates"].([]interface{})
	if !ok || len(templates) == 0 {
		t.Fatal("templates is empty or not a list")
	}
	tmpl, ok := templates[0].(map[string]interface{})
	if !ok {
		t.Fatal("template is not a map")
	}
	if tmpl["name"] != "ralph-executor" {
		t.Errorf("template name = %v, want ralph-executor", tmpl["name"])
	}

	container, ok := tmpl["container"].(map[string]interface{})
	if !ok {
		t.Fatal("container is not a map")
	}
	if container["image"] != "my-registry/ralph:v1.0.0" {
		t.Errorf("container.image = %v, want my-registry/ralph:v1.0.0", container["image"])
	}
	if container["workingDir"] != "/workspace" {
		t.Errorf("container.workingDir = %v, want /workspace", container["workingDir"])
	}

	env, ok := container["env"].([]interface{})
	if !ok {
		t.Fatal("env is not a list")
	}

	hasGitRepoURL, hasGitBranch, hasProjectBranch, hasCustomEnv, hasBaseBranch := false, false, false, false, false
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
			if envMap["value"] != cloneBranch {
				t.Errorf("GIT_BRANCH = %v, want %v", envMap["value"], cloneBranch)
			}
		case "PROJECT_BRANCH":
			hasProjectBranch = true
			if envMap["value"] != projectBranch {
				t.Errorf("PROJECT_BRANCH = %v, want %v", envMap["value"], projectBranch)
			}
		case "MY_VAR":
			if envMap["value"] == "my-value" {
				hasCustomEnv = true
			}
		case "BASE_BRANCH":
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

	volumeMounts, ok := container["volumeMounts"].([]interface{})
	if !ok {
		t.Fatal("volumeMounts is not a list")
	}

	hasGithubMount, hasOpencodeMount, hasConfigMapMount, hasSecretMount := false, false, false, false
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
		case mountMap["name"] == "my-config" && mountMap["mountPath"] == "/configmaps/my-config":
			hasConfigMapMount = true
		case mountMap["name"] == "my-secret" && mountMap["mountPath"] == "/secrets/my-secret":
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

	volumes, ok := tmpl["volumes"].([]interface{})
	if !ok {
		t.Fatal("volumes is not a list")
	}

	hasGithubVolume, hasOpencodeVolume, hasConfigMapVolume, hasSecretVolume := false, false, false, false
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
		case "my-config":
			hasConfigMapVolume = true
		case "my-secret":
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

	synchronization, ok := spec["synchronization"].(map[string]interface{})
	if !ok {
		t.Fatal("synchronization is not a map")
	}
	mutexes, ok := synchronization["mutexes"].([]interface{})
	if !ok {
		t.Fatal("mutexes is not a slice")
	}
	if len(mutexes) == 0 {
		t.Fatal("mutexes slice is empty")
	}
	mutex, ok := mutexes[0].(map[string]interface{})
	if !ok {
		t.Fatal("mutex is not a map")
	}
	mutexName, ok := mutex["name"].(string)
	if !ok {
		t.Fatal("mutex name is not a string")
	}
	expectedMutexName := "test-project"
	if mutexName != expectedMutexName {
		t.Errorf("mutex name = %v, want %v", mutexName, expectedMutexName)
	}
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

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	ctx := &execcontext.Context{ProjectFile: filepath.Join(tmpDir, "project.yaml")}
	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "project.yaml", false, false)
	if err != nil {
		t.Fatalf("GenerateWorkflowWithGitInfo failed: %v", err)
	}
	workflowYAML, err := wf.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var workflow map[string]interface{}
	if err := yaml.Unmarshal([]byte(workflowYAML), &workflow); err != nil {
		t.Fatalf("Failed to parse generated workflow YAML: %v", err)
	}

	spec := workflow["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	expectedImage := fmt.Sprintf("ghcr.io/zon/ralph:%s", DefaultContainerVersion())
	if container["image"] != expectedImage {
		t.Errorf("container.image = %v, want %v", container["image"], expectedImage)
	}
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

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	prBranch := "ralph/test-project"

	mw, err := GenerateMergeWorkflowWithGitInfo(repoURL, cloneBranch, prBranch)
	if err != nil {
		t.Fatalf("GenerateMergeWorkflowWithGitInfo failed: %v", err)
	}
	workflowYAML, err := mw.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var wf map[string]interface{}
	if err := yaml.Unmarshal([]byte(workflowYAML), &wf); err != nil {
		t.Fatalf("Failed to parse generated workflow YAML: %v", err)
	}

	if wf["apiVersion"] != "argoproj.io/v1alpha1" {
		t.Errorf("apiVersion = %v, want argoproj.io/v1alpha1", wf["apiVersion"])
	}
	if wf["kind"] != "Workflow" {
		t.Errorf("kind = %v, want Workflow", wf["kind"])
	}

	metadata, ok := wf["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata is not a map")
	}
	if metadata["generateName"] != "ralph-merge-" {
		t.Errorf("generateName = %v, want ralph-merge-", metadata["generateName"])
	}

	spec, ok := wf["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("spec is not a map")
	}
	if spec["entrypoint"] != "ralph-merger" {
		t.Errorf("entrypoint = %v, want ralph-merger", spec["entrypoint"])
	}

	ttlStrategy, ok := spec["ttlStrategy"].(map[string]interface{})
	if !ok {
		t.Fatal("ttlStrategy is not a map")
	}
	if ttlStrategy["secondsAfterCompletion"] != 86400 {
		t.Errorf("ttlStrategy.secondsAfterCompletion = %v, want 86400", ttlStrategy["secondsAfterCompletion"])
	}

	podGC, ok := spec["podGC"].(map[string]interface{})
	if !ok {
		t.Fatal("podGC is not a map")
	}
	if podGC["strategy"] != "OnWorkflowCompletion" {
		t.Errorf("podGC.strategy = %v, want OnWorkflowCompletion", podGC["strategy"])
	}

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

	container, ok := tmpl["container"].(map[string]interface{})
	if !ok {
		t.Fatal("container is not a map")
	}
	if container["image"] != "my-registry/ralph:v2.0.0" {
		t.Errorf("container.image = %v, want my-registry/ralph:v2.0.0", container["image"])
	}

	env, ok := container["env"].([]interface{})
	if !ok {
		t.Fatal("env is not a list")
	}

	hasPRBranch, hasGitRepoURL, hasGitBranch := false, false, false
	for _, e := range env {
		em, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		switch em["name"] {
		case "PR_BRANCH":
			hasPRBranch = true
			if em["value"] != prBranch {
				t.Errorf("PR_BRANCH = %v, want %v", em["value"], prBranch)
			}
		case "GIT_REPO_URL":
			hasGitRepoURL = true
		case "GIT_BRANCH":
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

	synchronization, ok := spec["synchronization"].(map[string]interface{})
	if !ok {
		t.Fatal("synchronization is not a map")
	}
	mutexes, ok := synchronization["mutexes"].([]interface{})
	if !ok {
		t.Fatal("mutexes is not a slice")
	}
	if len(mutexes) == 0 {
		t.Fatal("mutexes slice is empty")
	}
	mutex, ok := mutexes[0].(map[string]interface{})
	if !ok {
		t.Fatal("mutex is not a map")
	}
	mutexName, ok := mutex["name"].(string)
	if !ok {
		t.Fatal("mutex name is not a string")
	}
	expectedMutexName := "ralph-test-project"
	if mutexName != expectedMutexName {
		t.Errorf("mutex name = %v, want %v", mutexName, expectedMutexName)
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

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	mw, err := GenerateMergeWorkflowWithGitInfo("git@github.com:test/repo.git", "main", "ralph/test")
	if err != nil {
		t.Fatalf("GenerateMergeWorkflowWithGitInfo failed: %v", err)
	}
	workflowYAML, err := mw.Render()
	if err != nil {
		t.Fatalf("Render failed: %v", err)
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
	wf := &Workflow{RalphConfig: &config.RalphConfig{}}

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)
	os.Setenv("PATH", "")

	_, err := wf.Submit("test-namespace")
	if err == nil {
		t.Error("Expected error when argo CLI is not installed, got nil")
	}
	if !strings.Contains(err.Error(), "argo CLI not found") {
		t.Errorf("Error message should mention argo CLI not found, got: %v", err)
	}
}
