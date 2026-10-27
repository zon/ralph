package cmd

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/argo"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/output"
)

// TestPromptWorkflowClientSubmit covers generating and submitting a remote
// prompt workflow with the resolved values, the configured workflow options,
// and the resolved Kubernetes target.
func TestPromptWorkflowClientSubmit(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte(`
workflow:
  image:
    repository: registry.example/ralph
    tag: v1
  labels:
    team: platform
  context: config-context
  namespace: config-namespace
`), 0644))
	t.Chdir(dir)

	var submittedYAML string
	var submittedCtx argo.K8sContext
	argoClient := &argo.MockClient{
		SubmitYAMLFunc: func(_ context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submittedYAML = workflowYAML
			submittedCtx = kubeCtx
			return "ralph-prompt-abcde", nil
		},
	}

	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(io.Discard, io.Discard, false))
	ctx.SetRepoOwner("owner")
	ctx.SetRepoName("repo")
	ctx.SetDebugBranch("feature")
	ctx.SetModel("gpt-4")
	ctx.SetVariant("high")
	ctx.SetAgent("build")
	ctx.SetNoServices(true)
	ctx.SetKubeContext("override-context")
	ctx.SetKubeNamespace("override-namespace")

	client := &promptWorkflowClient{ctx: ctx, argoClient: argoClient}
	name, err := client.Submit(promptRemoteSubmission{
		CloneBranch:  "main",
		PromptBranch: "my-feature",
		BaseBranch:   "main",
		Prompt:       "Add the endpoint.",
	})
	require.NoError(t, err)
	assert.Equal(t, "ralph-prompt-abcde", name)
	assert.Equal(t, "override-context", submittedCtx.Name)
	assert.Equal(t, "override-namespace", submittedCtx.Namespace)

	var data map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(submittedYAML), &data))

	metadata := data["metadata"].(map[string]interface{})
	assert.True(t, strings.HasPrefix(metadata["generateName"].(string), "ralph-prompt-"))
	assert.Equal(t, "ralph", metadata["labels"].(map[string]interface{})["app.kubernetes.io/managed-by"])
	assert.Equal(t, "platform", metadata["labels"].(map[string]interface{})["team"])

	spec := data["spec"].(map[string]interface{})
	mutexes := spec["synchronization"].(map[string]interface{})["mutexes"].([]interface{})
	require.Len(t, mutexes, 1)
	assert.Equal(t, "my-feature", mutexes[0].(map[string]interface{})["name"])

	templates := spec["templates"].([]interface{})
	container := templates[0].(map[string]interface{})["container"].(map[string]interface{})
	assert.Equal(t, "registry.example/ralph:v1", container["image"])
	args := container["args"].([]interface{})
	assert.Contains(t, args, "Add the endpoint.")
	assert.Equal(t, "feature", promptArgValue(args, "--debug"))
	assert.Equal(t, "gpt-4", promptArgValue(args, "--model"))
	assert.Equal(t, "high", promptArgValue(args, "--variant"))
	assert.Equal(t, "build", promptArgValue(args, "--agent"))
	assert.Contains(t, args, "--no-services")
}

// TestPromptWorkflowClientPrintsHintAndFollows covers the log hint and followed
// logs using the submitted workflow's namespace and context.
// promptArgValue returns the value following the named argument, or "".
func promptArgValue(args []interface{}, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			if value, ok := args[i+1].(string); ok {
				return value
			}
		}
	}
	return ""
}

func TestPromptWorkflowClientPrintsHintAndFollows(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("workflow:\n  context: config-context\n  namespace: config-namespace\n"), 0644))
	t.Chdir(dir)

	var logsCtx argo.K8sContext
	var follow bool
	argoClient := &argo.MockClient{
		SubmitYAMLFunc: func(context.Context, string, argo.K8sContext) (string, error) {
			return "ralph-prompt-abcde", nil
		},
		LogsFunc: func(kubeCtx argo.K8sContext, _ string, followLogs bool) error {
			logsCtx = kubeCtx
			follow = followLogs
			return nil
		},
	}

	var out strings.Builder
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(&out, &out, false))
	ctx.SetRepoOwner("owner")
	ctx.SetRepoName("repo")

	client := &promptWorkflowClient{ctx: ctx, argoClient: argoClient}
	name, err := client.Submit(promptRemoteSubmission{CloneBranch: "main", PromptBranch: "my-feature", BaseBranch: "main", Prompt: "do work"})
	require.NoError(t, err)

	client.PrintLogHint(name)
	assert.Contains(t, out.String(), "argo logs -n config-namespace ralph-prompt-abcde -f")

	require.NoError(t, client.FollowLogs(name))
	assert.Equal(t, "config-context", logsCtx.Name)
	assert.Equal(t, "config-namespace", logsCtx.Namespace)
	assert.True(t, follow)
	assert.Contains(t, out.String(), "ralph-prompt-abcde", "the submitted workflow name is printed")
}

// TestPromptWorkflowClientFollowLeavesWorkflowRunning covers the detached
// workflow: a normal follow and an interrupted follow both leave the submitted
// workflow running instead of stopping it.
func TestPromptWorkflowClientFollowLeavesWorkflowRunning(t *testing.T) {
	tests := []struct {
		name      string
		followErr error
	}{
		{name: "completed follow", followErr: nil},
		{name: "interrupted follow", followErr: errors.New("signal: interrupt")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("workflow:\n  context: config-context\n  namespace: config-namespace\n"), 0644))
			t.Chdir(dir)

			stopped := false
			argoClient := &argo.MockClient{
				SubmitYAMLFunc: func(context.Context, string, argo.K8sContext) (string, error) {
					return "ralph-prompt-abcde", nil
				},
				LogsFunc: func(argo.K8sContext, string, bool) error {
					return tt.followErr
				},
				StopWorkflowFunc: func(argo.K8sContext, string) error {
					stopped = true
					return nil
				},
			}

			var out strings.Builder
			ctx := execcontext.NewContext()
			ctx.SetOutput(output.NewClient(&out, &out, false))
			ctx.SetRepoOwner("owner")
			ctx.SetRepoName("repo")

			client := &promptWorkflowClient{ctx: ctx, argoClient: argoClient}
			name, err := client.Submit(promptRemoteSubmission{CloneBranch: "main", PromptBranch: "my-feature", BaseBranch: "main", Prompt: "do work"})
			require.NoError(t, err)

			err = client.FollowLogs(name)
			if tt.followErr != nil {
				require.ErrorIs(t, err, tt.followErr)
			} else {
				require.NoError(t, err)
			}
			assert.False(t, stopped, "following logs must not stop the workflow")
		})
	}
}
