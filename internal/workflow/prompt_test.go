package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	githubpkg "github.com/zon/ralph/internal/github"
	"gopkg.in/yaml.v3"
)

// promptWorkflowData renders a prompt workflow and returns the parsed manifest.
func promptWorkflowData(t *testing.T, wf *Workflow) map[string]interface{} {
	t.Helper()
	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var data map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &data), "Failed to parse workflow YAML")
	return data
}

// promptContainerArgs renders a prompt workflow and returns the container args
// of the ralph-executor template.
func promptContainerArgs(t *testing.T, wf *Workflow) []interface{} {
	t.Helper()
	data := promptWorkflowData(t, wf)
	spec := data["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	return container["args"].([]interface{})
}

// TestGeneratePromptWorkflow asserts the generator carries the locally
// resolved branch, base branch, and prompt text into a prompt workflow.
func TestGeneratePromptWorkflow(t *testing.T) {
	ctx := &execcontext.Context{}
	ctx.SetModel("gpt-4")
	ctx.SetVariant("high")
	ctx.SetAgent("build")
	ctx.SetNoServices(true)
	ctx.SetDebugBranch("feature")
	ctx.SetVerbose(true)

	wf, err := GeneratePromptWorkflow(ctx, "my-feature", "main", "Add the endpoint.", "main", "git@github.com:test/repo.git")
	require.NoError(t, err, "GeneratePromptWorkflow failed")

	assert.Equal(t, "prompt", wf.ProjectName)
	assert.Equal(t, "test", wf.Repo.Owner)
	assert.Equal(t, "repo", wf.Repo.Name)
	assert.Equal(t, "main", wf.CloneBranch)
	assert.Equal(t, "my-feature", wf.ProjectBranch)
	assert.Equal(t, "main", wf.BaseBranch)
	require.NotNil(t, wf.Prompt)
	assert.Equal(t, "my-feature", wf.Prompt.Branch)
	assert.Equal(t, "Add the endpoint.", wf.Prompt.Text)
	assert.Equal(t, "gpt-4", wf.Model)
	assert.Equal(t, "high", wf.Variant)
	assert.Equal(t, "build", wf.Agent)
	assert.True(t, wf.NoServices)
	assert.Equal(t, "feature", wf.DebugBranch)
	assert.True(t, wf.Verbose)
}

// TestWorkflowRender_Prompt asserts the prompt workflow is named and labeled
// for Ralph, uses the same TTL and pod garbage collection as the other
// workflows, takes the prompt branch lock, and passes the prompt as one
// container argument.
func TestWorkflowRender_Prompt(t *testing.T) {
	wf := &Workflow{
		ProjectName:   "prompt",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "my-feature",
		BaseBranch:    "main",
		Prompt:        &PromptSpec{Branch: "my-feature", Text: "Add the endpoint."},
	}

	data := promptWorkflowData(t, wf)

	metadata := data["metadata"].(map[string]interface{})
	generateName := metadata["generateName"].(string)
	assert.True(t, strings.HasPrefix(generateName, "ralph-prompt-"), "generateName = %q, want prefix ralph-prompt-", generateName)
	labels := metadata["labels"].(map[string]interface{})
	assert.Equal(t, "ralph", labels["app.kubernetes.io/managed-by"])

	spec := data["spec"].(map[string]interface{})
	ttl := spec["ttlStrategy"].(map[string]interface{})
	assert.Equal(t, 86400, ttl["secondsAfterCompletion"])
	podGC := spec["podGC"].(map[string]interface{})
	assert.Equal(t, "OnWorkflowCompletion", podGC["strategy"])
	assert.Equal(t, "10m", podGC["deleteDelayDuration"])

	synchronization := spec["synchronization"].(map[string]interface{})
	mutexes := synchronization["mutexes"].([]interface{})
	require.Len(t, mutexes, 1)
	mutex := mutexes[0].(map[string]interface{})
	assert.Equal(t, "my-feature", mutex["name"], "the prompt branch lock is taken")

	args := promptContainerArgs(t, wf)
	assert.Equal(t, "workflow", args[0])
	assert.Equal(t, "prompt", args[1])
	assert.Equal(t, "main", argValue(args, "--base"))
	assert.Equal(t, "my-feature", argValue(args, "--branch"))
	assert.Equal(t, "Add the endpoint.", argValue(args, "--prompt"))

	promptArgs := argsForFlag(args, "--prompt")
	require.Len(t, promptArgs, 1, "the prompt is passed as one container argument")
	assert.Equal(t, "Add the endpoint.", promptArgs[0])
}

// TestWorkflowRender_PromptCarriesOptions asserts the prompt workflow passes
// the configured model, variant, agent, services, verbose, and debug choices to
// the container command.
func TestWorkflowRender_PromptCarriesOptions(t *testing.T) {
	wf := &Workflow{
		ProjectName:   "prompt",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "my-feature",
		BaseBranch:    "main",
		Prompt:        &PromptSpec{Branch: "my-feature", Text: "Add the endpoint."},
		Model:         "gpt-4",
		Variant:       "high",
		Agent:         "build",
		NoServices:    true,
		Verbose:       true,
		DebugBranch:   "feature",
	}

	args := promptContainerArgs(t, wf)

	assert.Equal(t, "gpt-4", argValue(args, "--model"))
	assert.Equal(t, "high", argValue(args, "--variant"))
	assert.Equal(t, "build", argValue(args, "--agent"))
	assert.Contains(t, args, "--no-services")
	assert.Contains(t, args, "--verbose")
	assert.Equal(t, "feature", argValue(args, "--debug"))
}

// TestWorkflowRender_PromptLeadingDash asserts a prompt that starts with a dash
// stays one container argument and cannot be mistaken for a flag.
func TestWorkflowRender_PromptLeadingDash(t *testing.T) {
	wf := &Workflow{
		ProjectName:   "prompt",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "my-feature",
		BaseBranch:    "main",
		Prompt:        &PromptSpec{Branch: "my-feature", Text: "--version"},
	}

	args := promptContainerArgs(t, wf)
	assert.Contains(t, args, "--prompt=--version")
	assert.NotContains(t, args, "--version", "the leading-dash prompt is not a standalone flag")
}

// TestGeneratePromptWorkflowConfiguredOptions asserts the prompt workflow uses
// the configured image, mounts, environment, labels, resources, context, and
// namespace.
func TestGeneratePromptWorkflowConfiguredOptions(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte(`
workflow:
  image:
    repository: registry.example/ralph
    tag: v1
  configMaps:
    - name: my-config
  secrets:
    - name: my-secret
  env:
    MY_VAR: my-value
  labels:
    team: platform
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
  context: prod-context
  namespace: prod-namespace
`), 0644))
	t.Chdir(dir)

	ctx := &execcontext.Context{}
	wf, err := GeneratePromptWorkflow(ctx, "my-feature", "main", "Add the endpoint.", "main", "git@github.com:test/repo.git")
	require.NoError(t, err)

	assert.Equal(t, "registry.example/ralph", wf.Image.Repository)
	assert.Equal(t, "v1", wf.Image.Tag)
	assert.Equal(t, "prod-context", wf.KubeContext)
	assert.Equal(t, "prod-namespace", wf.Namespace)
	assert.Equal(t, map[string]string{"team": "platform"}, wf.Labels)
	assert.Equal(t, "my-value", wf.Env["MY_VAR"].Value)
	assert.Equal(t, config.ResourceList{CPU: "100m", Memory: "128Mi"}, wf.Resources.Requests)
	assert.Equal(t, config.ResourceList{CPU: "500m", Memory: "512Mi"}, wf.Resources.Limits)
	require.Len(t, wf.ConfigMaps, 1)
	require.Len(t, wf.Secrets, 1)

	data := promptWorkflowData(t, wf)
	spec := data["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	assert.Equal(t, "registry.example/ralph:v1", container["image"])
	require.NotEmpty(t, container["resources"])
	assert.Equal(t, map[string]interface{}{"team": "platform"}, tmpl["metadata"].(map[string]interface{})["labels"])
}
