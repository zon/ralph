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
// on a main branch with the supplied resolved item query and cleanup setting,
// and returns the container args of the ralph-executor template.
func renderRemoteRunArgs(t *testing.T, items string, cleanup bool) []interface{} {
	t.Helper()
	cfg := &config.RalphConfig{
		DefaultBranch: "main",
		Workflow: config.WorkflowConfig{
			Namespace: "my-namespace",
		},
	}
	ctx := &execcontext.Context{}
	wf, err := GenerateWorkflowWithGitInfo(ctx, "test-project", "git@github.com:test/repo.git", "main", "test-project", "main", items, cleanup, "project.yaml", false, cfg, "")
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
	args := renderRemoteRunArgs(t, ".requirements", false)

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
	args := renderRemoteRunArgs(t, "", false)

	// THEN the container args include `--items .`, so the container does not
	// re-resolve the query
	require.Equal(t, ".", itemsArgValue(args))
}

// TestRemoteArgumentsCleanupEnabledScenario covers the "Cleanup enabled"
// scenario: when cleanup resolved to enabled before workflow submission, the
// container args for `ralph workflow run` include `--cleanup`.
func TestRemoteArgumentsCleanupEnabledScenario(t *testing.T) {
	// GIVEN cleanup resolved to enabled before workflow submission
	// WHEN the workflow YAML is generated
	args := renderRemoteRunArgs(t, ".requirements", true)

	// THEN the container args for `ralph workflow run` include `--cleanup`
	assert.Contains(t, args, "--cleanup")
}

// TestRemoteArgumentsCleanupDisabledScenario covers the "Cleanup disabled"
// scenario: when cleanup resolved to disabled, the container args contain no
// `--cleanup` flag.
func TestRemoteArgumentsCleanupDisabledScenario(t *testing.T) {
	// GIVEN cleanup resolved to disabled
	// WHEN the workflow YAML is generated
	args := renderRemoteRunArgs(t, ".requirements", false)

	// THEN the container args contain no `--cleanup` flag
	assert.NotContains(t, args, "--cleanup")
}

// TestRemoteArgumentsItemQueryAlwaysPassedExplicitly covers the item that the
// resolved item query is always passed explicitly to the container: the
// manifest carries a `--items` argument even when the query falls back to the
// default `.`, so the container never re-resolves it from the repository config.
func TestRemoteArgumentsItemQueryAlwaysPassedExplicitly(t *testing.T) {
	args := renderRemoteRunArgs(t, "", true)
	require.Equal(t, ".", itemsArgValue(args), "--items must always be present with the resolved query")

	args = renderRemoteRunArgs(t, ".requirements", false)
	require.Equal(t, ".requirements", itemsArgValue(args))
}

// TestRemoteArgumentsCleanupOnlyWhenEnabled covers the item that the cleanup
// flag appears in the container args only when cleanup is enabled.
func TestRemoteArgumentsCleanupOnlyWhenEnabled(t *testing.T) {
	assert.NotContains(t, renderRemoteRunArgs(t, ".requirements", false), "--cleanup")
	assert.Contains(t, renderRemoteRunArgs(t, ".requirements", true), "--cleanup")
}
