package workflow

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	githubpkg "github.com/zon/ralph/internal/github"
	"gopkg.in/yaml.v3"
)

// ConfigLoader is the interface for loading ralph configuration.
type ConfigLoader interface {
	LoadConfig() (*config.RalphConfig, error)
}

// mockConfig implements ConfigLoader using the function-field pattern from docs/testing.md.
type mockConfig struct {
	loadConfigFn func() (*config.RalphConfig, error)
}

func (m *mockConfig) LoadConfig() (*config.RalphConfig, error) {
	if m.loadConfigFn != nil {
		return m.loadConfigFn()
	}
	return &config.RalphConfig{}, nil
}

// FileSystem is the interface for filesystem operations.
type FileSystem interface {
	Getwd() (string, error)
	ReadFile(string) ([]byte, error)
}

// mockFileSystem implements FileSystem using the function-field pattern.
type mockFileSystem struct {
	getwdFn    func() (string, error)
	readFileFn func(string) ([]byte, error)
}

func (m *mockFileSystem) Getwd() (string, error) {
	if m.getwdFn != nil {
		return m.getwdFn()
	}
	return "/tmp", nil
}

func (m *mockFileSystem) ReadFile(path string) ([]byte, error) {
	if m.readFileFn != nil {
		return m.readFileFn(path)
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

// GitClient is the interface for git operations.
type GitClient interface {
	FindRepoRoot() (string, error)
	RemoteURL() (string, error)
	CurrentBranch() (string, error)
}

// mockGit implements GitClient using the function-field pattern.
type mockGit struct {
	findRepoRootFn    func() (string, error)
	remoteURLFn       func() (string, error)
	currentBranchFn   func() (string, error)
}

func (m *mockGit) FindRepoRoot() (string, error) {
	if m.findRepoRootFn != nil {
		return m.findRepoRootFn()
	}
	return "/tmp/repo", nil
}

func (m *mockGit) RemoteURL() (string, error) {
	if m.remoteURLFn != nil {
		return m.remoteURLFn()
	}
	return "git@github.com:owner/repo.git", nil
}

func (m *mockGit) CurrentBranch() (string, error) {
	if m.currentBranchFn != nil {
		return m.currentBranchFn()
	}
	return "main", nil
}

// GitHubClient is the interface for GitHub operations.
type GitHubClient interface {
	GetRepo() (githubpkg.Repo, error)
	ParseRemoteURL(string) (githubpkg.Repo, error)
}

// mockGitHub implements GitHubClient using the function-field pattern.
type mockGitHub struct {
	getRepoFn        func() (githubpkg.Repo, error)
	parseRemoteURLFn func(string) (githubpkg.Repo, error)
}

func (m *mockGitHub) GetRepo() (githubpkg.Repo, error) {
	if m.getRepoFn != nil {
		return m.getRepoFn()
	}
	return githubpkg.Repo{}, nil
}

func (m *mockGitHub) ParseRemoteURL(url string) (githubpkg.Repo, error) {
	if m.parseRemoteURLFn != nil {
		return m.parseRemoteURLFn(url)
	}
	return githubpkg.ParseRemoteURL(url)
}

func TestGenerateWorkflow(t *testing.T) {
	ctx := &execcontext.Context{}
	ctx.SetNoServices(true)

	cfg := &config.RalphConfig{
		DefaultBranch: "main",
		Workflow: config.WorkflowConfig{
			Image: config.ImageConfig{
				Repository: "my-registry/ralph",
				Tag:        "v1.0.0",
			},
			ConfigMaps: []config.ConfigMapMount{
				{Name: "my-config"},
			},
			Secrets: []config.SecretMount{
				{Name: "my-secret"},
			},
			Env: map[string]string{
				"MY_VAR": "my-value",
			},
			Context:   "my-context",
			Namespace: "my-namespace",
		},
	}
	instructions := "# Custom Instructions\n\nTest instructions"

	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	projectBranch := "test-project"
	relProjectPath := "project.yaml"

	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, cloneBranch, projectBranch, relProjectPath, false, cfg, instructions)
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

	labels, ok := metadata["labels"].(map[string]interface{})
	require.True(t, ok, "metadata labels is not a map")
	assert.Equal(t, "ralph", labels["app.kubernetes.io/managed-by"], "workflow metadata should contain app.kubernetes.io/managed-by=ralph")

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
	assert.Equal(t, []interface{}{"ralph"}, container["command"])
	assert.Equal(t, []interface{}{"workflow", "run", "--repo", "test/repo", "--project-path", "{{workflow.parameters.project-path}}", "--project-branch", projectBranch, "--base", "main", "--no-services"}, container["args"])

	env, ok := container["env"].([]interface{})
	require.True(t, ok, "env is not a list")

	hasGitRepoURL, hasGitBranch, hasProjectBranch, hasCustomEnv, hasBaseBranch, hasPulumiToken, hasDebugBranch, hasVerbose, hasNoServices := false, false, false, false, false, false, false, false, false
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
		case "RALPH_DEBUG_BRANCH":
			hasDebugBranch = true
		case "RALPH_VERBOSE":
			hasVerbose = true
		case "RALPH_NO_SERVICES":
			hasNoServices = true
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
	assert.True(t, hasDebugBranch, "RALPH_DEBUG_BRANCH environment variable not found")
	assert.True(t, hasVerbose, "RALPH_VERBOSE environment variable not found")
	assert.True(t, hasNoServices, "RALPH_NO_SERVICES environment variable not found")

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
	cfg := &config.RalphConfig{
		DefaultBranch: "main",
	}
	ctx := &execcontext.Context{}
	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "project.yaml", false, cfg, "")
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
	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	prBranch := "ralph/test-project"

	mw, err := GenerateMergeWorkflowWithGitInfo(repoURL, cloneBranch, prBranch, "", WorkflowOptions{
		Image: MakeImage("my-registry/ralph", "v2.0.0"),
	})
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

	labels, ok := metadata["labels"].(map[string]interface{})
	require.True(t, ok, "metadata labels is not a map")
	assert.Equal(t, "ralph", labels["app.kubernetes.io/managed-by"], "workflow metadata should contain app.kubernetes.io/managed-by=ralph")

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
	mw, err := GenerateMergeWorkflowWithGitInfo("git@github.com:test/repo.git", "main", "ralph/test", "", WorkflowOptions{})
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
	wf := &Workflow{}

	t.Setenv("PATH", "")

	client := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			return "", fmt.Errorf("argo CLI not found")
		},
	}
	_, err := wf.Submit(context.Background(), client)
	require.Error(t, err, "Expected error when argo CLI is not installed")
	assert.True(t, strings.Contains(err.Error(), "argo CLI not found"), "Error message should mention argo CLI not found, got: %v", err)
}

func TestWorkflowRender_CommentBranching(t *testing.T) {
	commentBody := "Please review this PR"
	prNumber := "123"

	wf := &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		CommentBody:   commentBody,
		PRNumber:      prNumber,
	}

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	assert.True(t, strings.Contains(workflowYAML, commentBody), "Rendered YAML should contain comment body %q", commentBody)

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	command := container["command"].([]interface{})
	args := container["args"].([]interface{})

	assert.Equal(t, "ralph", command[0], "Command should be 'ralph' for comment workflow")
	assert.Equal(t, "workflow", args[0], "First arg should be 'workflow'")
	assert.Equal(t, "comment", args[1], "Second arg should be 'comment'")
	assert.Equal(t, "--comment-body", args[8], "Should have --comment-body flag")
	assert.Equal(t, commentBody, args[9], "Comment body should be passed as arg")
	assert.Equal(t, "--pr", args[10], "Should have --pr flag")
}

func TestWorkflowRender_RunBranching(t *testing.T) {
	wf := &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		CommentBody:   "",
		PRNumber:      "",
	}

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	command := container["command"].([]interface{})
	args := container["args"].([]interface{})

	assert.Equal(t, "ralph", command[0], "Command should be 'ralph' for regular workflow")
	assert.Equal(t, "workflow", args[0], "First arg should be 'workflow' for regular workflow")
	assert.Equal(t, "run", args[1], "Second arg should be 'run' for regular workflow")
	assert.Equal(t, "--repo", args[2], "Third arg should be '--repo'")
}

func TestWorkflowRender_DebugBranch(t *testing.T) {
	debugBranch := "feat/debug-mode"
	wf := &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		DebugBranch:   debugBranch,
	}

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	command := container["command"].([]interface{})
	args := container["args"].([]interface{})

	assert.Equal(t, "ralph", command[0], "Command should be 'ralph' for debug workflow")
	assert.Equal(t, "run", args[1], "Second arg should be 'run'")
	assert.Equal(t, "--debug", args[10], "Should have --debug flag")
	assert.Equal(t, debugBranch, args[11], "Debug branch should be passed as arg")

	env := container["env"].([]interface{})
	foundDebugBranch := false
	for _, e := range env {
		envVar := e.(map[string]interface{})
		if envVar["name"] == "RALPH_DEBUG_BRANCH" {
			assert.Equal(t, debugBranch, envVar["value"])
			foundDebugBranch = true
		}
	}
	assert.True(t, foundDebugBranch, "RALPH_DEBUG_BRANCH environment variable not found")
}

func TestWorkflowRender_WithLabels(t *testing.T) {
	labels := map[string]string{
		"environment":            "production",
		"team":                   "platform",
		"app.kubernetes.io/name": "ralph",
	}

	wf := &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		Labels:        labels,
	}

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})

	podMetadata := tmpl["metadata"].(map[string]interface{})
	podLabels := podMetadata["labels"].(map[string]interface{})

	assert.Equal(t, "production", podLabels["environment"])
	assert.Equal(t, "platform", podLabels["team"])
	assert.Equal(t, "ralph", podLabels["app.kubernetes.io/name"])
}

func TestMergeWorkflowRender_EnvVarCoverage(t *testing.T) {
	repoOwner := "test-owner"
	repoName := "test-repo"
	prNumber := "456"

	mw := &MergeWorkflow{
		Repo:        githubpkg.MakeRepo(repoOwner, repoName),
		CloneBranch: "main",
		PRBranch:    "feature-branch",
		PRNumber:    prNumber,
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
	mw := &MergeWorkflow{
		Repo:        githubpkg.MakeRepo("test-owner", "test-repo"),
		CloneBranch: "main",
		PRBranch:    "feature-branch",
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
	cfg := &config.RalphConfig{
		DefaultBranch: "config-base-branch",
		Workflow: config.WorkflowConfig{
			Namespace: "my-namespace",
		},
	}
	ctx := &execcontext.Context{}
	ctx.SetBaseBranch("override-branch")

	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	projectBranch := "test-project"
	relProjectPath := "project.yaml"

	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, cloneBranch, projectBranch, relProjectPath, false, cfg, "")
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
	cfg := &config.RalphConfig{
		DefaultBranch: "main",
		Workflow: config.WorkflowConfig{
			Namespace: "my-namespace",
		},
	}
	ctx := &execcontext.Context{}

	repoURL := "git@github.com:test/repo.git"
	cloneBranch := "main"
	projectBranch := "test-project"
	relProjectPath := "project.yaml"

	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", repoURL, cloneBranch, projectBranch, relProjectPath, false, cfg, "")
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

func TestKubeContextOverride(t *testing.T) {
	cfg := &config.RalphConfig{
		DefaultBranch: "main",
		Workflow: config.WorkflowConfig{
			Context:   "config-context",
			Namespace: "my-namespace",
		},
	}

	t.Run("context override takes precedence over config", func(t *testing.T) {
		ctx := &execcontext.Context{}
		ctx.SetKubeContext("override-context")

		wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "project.yaml", false, cfg, "")
		require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")

		assert.Equal(t, "override-context", wf.KubeContext, "KubeContext should be set from context override")
	})

	t.Run("falls back to config when context override is empty", func(t *testing.T) {
		ctx := &execcontext.Context{}

		wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "project.yaml", false, cfg, "")
		require.NoError(t, err)

		assert.Equal(t, "config-context", wf.KubeContext, "KubeContext should fall back to config")
	})
}

func TestWorkflowRender_CommandField(t *testing.T) {
	wf := &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		Command:       []string{"echo", "hello world"},
	}

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	command := container["command"].([]interface{})
	args := container["args"].([]interface{})

	assert.Equal(t, "ralph", command[0], "Command should be 'ralph'")
	assert.Equal(t, "workflow", args[0], "First arg should be 'workflow'")
	assert.Equal(t, "--command", args[1], "Second arg should be '--command'")
	assert.Equal(t, "--", args[2], "Third arg should be '--'")
	assert.Equal(t, "echo", args[3], "Fourth arg should be command token 'echo'")
	assert.Equal(t, "hello world", args[4], "Fifth arg should be command token 'hello world'")
}

func TestWorkflowRender_CommandFieldEmpty(t *testing.T) {
	wf := &Workflow{
		ProjectName:   "test-project",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "feature-branch",
		ProjectPath:   "project.yaml",
		Command:       nil,
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

	assert.NotContains(t, args, "--command", "Args should not contain --command when Command is empty/nil")
}

func TestGenerateCommandWorkflow(t *testing.T) {
	repo, err := githubpkg.ParseRemoteURL("git@github.com:testowner/testrepo.git")
	require.NoError(t, err)

	wf := &Workflow{
		ProjectName: "command",
		Repo:        repo,
		CloneBranch: "main",
		Command:     []string{"echo", "hello"},
		Verbose:     true,
		NoServices:  true,
	}

	assert.Equal(t, "command", wf.ProjectName)
	assert.Equal(t, "main", wf.CloneBranch)
	assert.Equal(t, []string{"echo", "hello"}, wf.Command)
	assert.True(t, wf.Verbose)
	assert.True(t, wf.NoServices)
	assert.Equal(t, "testowner", wf.Repo.Owner)
	assert.Equal(t, "testrepo", wf.Repo.Name)

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var workflow map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &workflow), "Failed to parse generated workflow YAML")

	assert.Equal(t, "argoproj.io/v1alpha1", workflow["apiVersion"])
	assert.Equal(t, "Workflow", workflow["kind"])

	metadata, ok := workflow["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata is not a map")
	generateName, ok := metadata["generateName"].(string)
	require.True(t, ok && strings.HasPrefix(generateName, "ralph-command-"), "generateName = %v, want prefix ralph-command-", generateName)

	labels, ok := metadata["labels"].(map[string]interface{})
	require.True(t, ok, "metadata labels is not a map")
	assert.Equal(t, "ralph", labels["app.kubernetes.io/managed-by"], "workflow metadata should contain app.kubernetes.io/managed-by=ralph")

	spec, ok := workflow["spec"].(map[string]interface{})
	require.True(t, ok, "spec is not a map")

	templates, ok := spec["templates"].([]interface{})
	require.True(t, ok && len(templates) > 0, "templates is empty or not a list")
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})

	command := container["command"].([]interface{})
	args := container["args"].([]interface{})

	assert.Equal(t, "ralph", command[0], "Command should be 'ralph'")
	assert.Equal(t, "workflow", args[0], "First arg should be 'workflow'")
	assert.Equal(t, "--command", args[1], "Second arg should be '--command'")
	assert.Equal(t, "--", args[2], "Third arg should be '--'")
	assert.Equal(t, "echo", args[3], "Fourth arg should be command token 'echo'")
	assert.Equal(t, "hello", args[4], "Fifth arg should be command token 'hello'")
	assert.Equal(t, "--verbose", args[5], "Sixth arg should be '--verbose'")
}
