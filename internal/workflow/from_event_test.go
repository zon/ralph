package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// workflowTestDir sets up a temp dir with the minimal .ralph/config.yaml that
// workflow generation functions may require, changes into it for the test, and
// restores the original working directory on cleanup.
func workflowTestDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	ralphDir := filepath.Join(tmp, ".ralph")
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("maxIterations: 3\n"), 0644))

	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(orig) })
}

func TestProjectFileFromBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   string
	}{
		{"ralph/my-feature", "projects/my-feature.yaml"},
		{"ralph/github-webhook-service", "projects/github-webhook-service.yaml"},
		{"feature/something", "projects/feature-something.yaml"},
		{"", "projects/.yaml"},
	}
	for _, tc := range tests {
		t.Run(tc.branch, func(t *testing.T) {
			assert.Equal(t, tc.want, ProjectFileFromBranch(tc.branch))
		})
	}
}

func TestFromWebhookEvent_CommentEvent_ReturnsRunWorkflow(t *testing.T) {
	workflowTestDir(t)

	opts := WorkflowOptions{}
	we := WebhookEvent{
		Body:      "please add a test",
		PRBranch:  "ralph/my-feature",
		PRNumber:  "5",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := FromWebhookEvent(we, opts)
	require.NoError(t, err)
	require.NotNil(t, result.Run)
	assert.Equal(t, "please add a test", result.Run.CommentBody)
	assert.Equal(t, "5", result.Run.PRNumber)
	assert.Equal(t, "acme", result.Run.Repo.Owner)
	assert.Equal(t, "myrepo", result.Run.Repo.Name)
	assert.Equal(t, "ralph/my-feature", result.Run.CloneBranch)

	workflowYAML, err := result.Run.Render()
	require.NoError(t, err)
	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData))

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	args := container["args"].([]interface{})

	assert.Equal(t, "ralph", container["command"].([]interface{})[0], "Command should be 'ralph' for a run workflow")
	assert.Equal(t, "workflow", args[0], "First arg should be 'workflow'")
	assert.Equal(t, "comment", args[1], "Second arg should be 'comment'. A webhook event only ever produces a run workflow")
}

func TestFromWebhookEvent_RunWorkflow_RendersToYAML(t *testing.T) {
	workflowTestDir(t)

	opts := WorkflowOptions{}
	we := WebhookEvent{
		Body:      "fix the bug",
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := FromWebhookEvent(we, opts)
	require.NoError(t, err)

	yaml, err := result.Run.Render()
	require.NoError(t, err)
	assert.Contains(t, yaml, "argoproj.io/v1alpha1")
	assert.Contains(t, yaml, "acme")
	assert.Contains(t, yaml, "myrepo")
	assert.Contains(t, yaml, "ralph/my-feature")
}

// Scenario: Run Workflow labeled.
//
// GIVEN a comment event triggers a Run Workflow submission
// WHEN the workflow YAML is rendered
// THEN the workflow metadata contains the label app.kubernetes.io/managed-by=ralph
func TestFromWebhookEvent_RunWorkflow_Labeled(t *testing.T) {
	workflowTestDir(t)

	opts := WorkflowOptions{}
	we := WebhookEvent{
		Body:      "fix the bug",
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
		PRNumber:  "7",
	}

	result, err := FromWebhookEvent(we, opts)
	require.NoError(t, err)
	require.NotNil(t, result.Run)

	workflowYAML, err := result.Run.Render()
	require.NoError(t, err)

	var workflow map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &workflow))

	metadata, ok := workflow["metadata"].(map[string]interface{})
	require.True(t, ok, "metadata is not a map")
	labels, ok := metadata["labels"].(map[string]interface{})
	require.True(t, ok, "metadata labels is not a map")
	assert.Equal(t, "ralph", labels["app.kubernetes.io/managed-by"], "workflow metadata should contain app.kubernetes.io/managed-by=ralph")
}

func TestFromWebhookEvent_NamespacePropagated(t *testing.T) {
	workflowTestDir(t)

	opts := WorkflowOptions{Namespace: "team-ns"}
	we := WebhookEvent{
		Body:      "fix something",
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}
	result, err := FromWebhookEvent(we, opts)
	require.NoError(t, err)
	assert.Equal(t, "team-ns", result.Namespace)
	require.NotNil(t, result.Run)
	assert.Equal(t, "team-ns", result.Run.Namespace)
}

func TestFromWebhookEvent_Namespace_EmptyWhenNotConfigured(t *testing.T) {
	workflowTestDir(t)

	opts := WorkflowOptions{}
	we := WebhookEvent{
		Body:      "fix something",
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := FromWebhookEvent(we, opts)
	require.NoError(t, err)
	assert.Equal(t, "", result.Namespace)
}
