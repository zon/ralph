package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"gopkg.in/yaml.v3"
)

// renderRemoteRunArgs generates a run workflow for a project named test-project
// on a main branch with the supplied resolved item query, and returns the
// container args of the ralph-executor template.
func renderRemoteRunArgs(t *testing.T, items string) []interface{} {
	t.Helper()
	cfg := &config.RalphConfig{
		DefaultBranch: "main",
		Workflow: config.WorkflowConfig{
			Namespace: "my-namespace",
		},
	}
	ctx := &execcontext.Context{}
	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "main", items, "project.yaml", false, cfg, "")
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")
	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var workflow map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &workflow), "Failed to parse generated workflow YAML")
	spec := workflow["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	require.NotEmpty(t, templates, "templates is empty")
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	return container["args"].([]interface{})
}

// itemsArgValue returns the value following the --items argument, or "" when
// the args contain no --items argument.
func itemsArgValue(args []interface{}) string {
	for i, a := range args {
		if a == "--items" && i+1 < len(args) {
			if v, ok := args[i+1].(string); ok {
				return v
			}
		}
	}
	return ""
}

// TestRemoteArgumentsResolvedItemsScenario covers the "Resolved item query
// passed as --items argument" scenario: when the item query resolved locally to
// `.requirements`, the workflow container args include `--items .requirements`.
func TestRemoteArgumentsResolvedItemsScenario(t *testing.T) {
	// GIVEN the item query resolved locally to .requirements
	// WHEN the workflow YAML is generated
	args := renderRemoteRunArgs(t, ".requirements")

	// THEN the container args for `ralph workflow run` include `--items .requirements`
	require.Equal(t, ".requirements", itemsArgValue(args))
}

// TestRemoteArgumentsDefaultQueryPassedExplicitlyScenario covers the "Default
// query passed explicitly" scenario: when neither `--items` nor `items` in
// `.ralph/config.yaml` is set, the query resolves to `.` and the container args
// include `--items .`, so the container does not re-resolve the query.
func TestRemoteArgumentsDefaultQueryPassedExplicitlyScenario(t *testing.T) {
	// GIVEN neither `--items` nor `items` in `.ralph/config.yaml` is set,
	// so the query resolves to `.`
	// WHEN the workflow YAML is generated
	args := renderRemoteRunArgs(t, "")

	// THEN the container args include `--items .`, so the container does not
	// re-resolve the query
	require.Equal(t, ".", itemsArgValue(args))
}

// TestRemoteArgumentsCleanupNotPlumbedScenario covers the item that project
// file cleanup is no longer delivered to the container as an argument: the
// `--cleanup` flag was removed when cleanup became the default, so the
// container args for `ralph workflow run` never include it. The container reads
// the `cleanup` field from the repository's own .ralph/config.yaml instead.
func TestRemoteArgumentsCleanupNotPlumbedScenario(t *testing.T) {
	// WHEN the workflow YAML is generated
	args := renderRemoteRunArgs(t, ".requirements")

	// THEN the container args contain no `--cleanup` flag
	assert.NotContains(t, args, "--cleanup")
}

// TestRemoteArgumentsItemQueryAlwaysPassedExplicitly covers the item that the
// resolved item query is always passed explicitly to the container: the
// manifest carries a `--items` argument even when the query falls back to the
// default `.`, so the container never re-resolves it from the repository config.
func TestRemoteArgumentsItemQueryAlwaysPassedExplicitly(t *testing.T) {
	args := renderRemoteRunArgs(t, "")
	require.Equal(t, ".", itemsArgValue(args), "--items must always be present with the resolved query")

	args = renderRemoteRunArgs(t, ".requirements")
	require.Equal(t, ".requirements", itemsArgValue(args))
}

// TestRemoteArgumentsAgentScenario covers the "Agent override" scenario: when
// an opencode agent is set on the context, the container args for
// `ralph workflow run` include `--agent <name>`.
func TestRemoteArgumentsAgentScenario(t *testing.T) {
	// GIVEN an agent override is set on the context
	ctx := &execcontext.Context{}
	ctx.SetAgent("code-reviewer")
	cfg := &config.RalphConfig{DefaultBranch: "main"}
	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "main", "", "project.yaml", false, cfg, "")
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")
	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var workflow map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &workflow), "Failed to parse generated workflow YAML")
	spec := workflow["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	require.NotEmpty(t, templates, "templates is empty")
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	args := container["args"].([]interface{})

	// THEN the container args for `ralph workflow run` include `--agent code-reviewer`
	assert.Contains(t, args, "--agent")
	assert.Contains(t, args, "code-reviewer")
}

// TestRemoteArgumentsAgentOmittedWhenUnset covers the item that the `--agent`
// flag appears in the container args only when an agent is set.
func TestRemoteArgumentsAgentOmittedWhenUnset(t *testing.T) {
	assert.NotContains(t, renderRemoteRunArgs(t, ".requirements"), "--agent")
}

// TestRemoteArgumentsVariantScenario covers the "Variant override" scenario:
// when a variant is set on the context, the container args for
// `ralph workflow run` include `--variant <hint>`.
func TestRemoteArgumentsVariantScenario(t *testing.T) {
	// GIVEN a variant override is set on the context
	ctx := &execcontext.Context{}
	ctx.SetVariant("high")
	cfg := &config.RalphConfig{DefaultBranch: "main"}
	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "main", "", "project.yaml", false, cfg, "")
	require.NoError(t, err, "GenerateWorkflowWithGitInfo failed")
	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var workflow map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &workflow), "Failed to parse generated workflow YAML")
	spec := workflow["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	require.NotEmpty(t, templates, "templates is empty")
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	args := container["args"].([]interface{})

	// THEN the container args for `ralph workflow run` include `--variant high`
	assert.Contains(t, args, "--variant")
	assert.Contains(t, args, "high")
}

// TestRemoteArgumentsVariantOmittedWhenUnset covers the item that the
// `--variant` flag appears in the container args only when a variant is set.
func TestRemoteArgumentsVariantOmittedWhenUnset(t *testing.T) {
	assert.NotContains(t, renderRemoteRunArgs(t, ".requirements"), "--variant")
}
