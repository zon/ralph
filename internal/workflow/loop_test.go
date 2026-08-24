package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	execcontext "github.com/zon/ralph/internal/context"
	githubpkg "github.com/zon/ralph/internal/github"
	"gopkg.in/yaml.v3"
)

// loopContainerArgs renders a workflow with the given loop spec and returns the
// container args of the ralph-executor template.
func loopContainerArgs(t *testing.T, loop *LoopSpec, verbose bool) []interface{} {
	t.Helper()
	wf := &Workflow{
		ProjectName:   "loop",
		Repo:          githubpkg.MakeRepo("owner", "repo"),
		CloneBranch:   "main",
		ProjectBranch: "loop-fmt",
		Loop:          loop,
		Verbose:       verbose,
	}
	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")

	var wfData map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData), "Failed to parse workflow YAML")

	spec := wfData["spec"].(map[string]interface{})
	templates := spec["templates"].([]interface{})
	tmpl := templates[0].(map[string]interface{})
	container := tmpl["container"].(map[string]interface{})
	return container["args"].([]interface{})
}

// argValue returns the string value following the named argument, or "" when
// the args contain no such argument.
func argValue(args []interface{}, flag string) string {
	values := argsForFlag(args, flag)
	if len(values) == 0 {
		return ""
	}
	if v, ok := values[0].(string); ok {
		return v
	}
	return ""
}

// argsForFlag returns the values immediately following every occurrence of the
// named argument. When the flag is absent it returns nil.
func argsForFlag(args []interface{}, flag string) []interface{} {
	var values []interface{}
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			values = append(values, args[i+1])
		}
	}
	return values
}

// TestWorkflowRender_Loop asserts a workflow carrying a Loop spec invokes
// `ralph workflow loop` with the slug, steps, max iterations, repo, clone
// branch, and bot identity.
func TestWorkflowRender_Loop(t *testing.T) {
	args := loopContainerArgs(t, &LoopSpec{Slug: "fmt", Steps: []string{"run gofmt", "run go vet"}, Max: 3}, false)

	assert.Equal(t, "workflow", args[0], "First arg should be 'workflow'")
	assert.Equal(t, "loop", args[1], "Second arg should be 'loop'")
	assert.Equal(t, "fmt", argValue(args, "--slug"), "The slug value is passed")
	assert.Equal(t, "owner/repo", argValue(args, "--repo"), "The repo value is passed")
	assert.Equal(t, "main", argValue(args, "--clone-branch"), "The clone branch value is passed")
	assert.Equal(t, "3", argValue(args, "--max"), "The max iterations value is passed")
	assert.Equal(t, "ralph-zon[bot]", argValue(args, "--bot-name"), "The bot name value is passed")
	assert.Equal(t, "ralph-zon[bot]@users.noreply.github.com", argValue(args, "--bot-email"), "The bot email value is passed")
	assert.Equal(t, []interface{}{"run gofmt", "run go vet"}, argsForFlag(args, "--step"), "The steps are passed in order as --step arguments")
}

// TestWorkflowRender_LoopWithoutSlug asserts a loop workflow omits the --slug
// argument when no slug is resolved, so the container proposes one from the
// steps.
func TestWorkflowRender_LoopWithoutSlug(t *testing.T) {
	args := loopContainerArgs(t, &LoopSpec{Steps: []string{"run gofmt"}, Max: 10}, false)

	assert.NotContains(t, args, "--slug", "No --slug argument is passed when the slug is empty")
	assert.Contains(t, args, "loop", "The loop subcommand is invoked")
}

// TestWorkflowRender_LoopOmitsMaxWhenZero asserts the --max argument is omitted
// when max is unset, leaving the container command's default in place.
func TestWorkflowRender_LoopOmitsMaxWhenZero(t *testing.T) {
	args := loopContainerArgs(t, &LoopSpec{Slug: "fmt", Steps: []string{"run gofmt"}}, false)

	assert.NotContains(t, args, "--max", "No --max argument is passed when max is zero")
}

// TestWorkflowRender_LoopVerbose asserts --verbose is passed to the container
// command when the workflow is rendered verbose.
func TestWorkflowRender_LoopVerbose(t *testing.T) {
	args := loopContainerArgs(t, &LoopSpec{Slug: "fmt", Steps: []string{"run gofmt"}, Max: 10}, true)

	assert.Contains(t, args, "--verbose", "The --verbose argument is passed when verbose")
}

// TestGenerateLoopWorkflow asserts GenerateLoopWorkflow carries the slug, steps,
// and max into the loop spec, parses the repo, and keeps the clone branch.
func TestGenerateLoopWorkflow(t *testing.T) {
	ctx := &execcontext.Context{}
	wf, err := GenerateLoopWorkflow(ctx, "fmt", []string{"run gofmt", "run go vet"}, 3, "main", "git@github.com:test/repo.git")
	require.NoError(t, err, "GenerateLoopWorkflow failed")

	assert.Equal(t, "loop", wf.ProjectName, "the workflow project name is loop")
	assert.Equal(t, "main", wf.CloneBranch, "the clone branch is kept")
	assert.Equal(t, "test", wf.Repo.Owner, "the repo owner is parsed")
	assert.Equal(t, "repo", wf.Repo.Name, "the repo name is parsed")
	require.NotNil(t, wf.Loop, "the loop spec is set")
	assert.Equal(t, "fmt", wf.Loop.Slug, "the slug is carried in the loop spec")
	assert.Equal(t, []string{"run gofmt", "run go vet"}, wf.Loop.Steps, "the steps are carried in the loop spec")
	assert.Equal(t, 3, wf.Loop.Max, "the max iterations are carried in the loop spec")
}

// TestGenerateLoopWorkflowWithoutSteps asserts a steps-only loop keeps an empty
// slug in the loop spec so the container proposes one.
func TestGenerateLoopWorkflowWithoutSteps(t *testing.T) {
	ctx := &execcontext.Context{}
	wf, err := GenerateLoopWorkflow(ctx, "", []string{"run gofmt"}, 10, "main", "git@github.com:test/repo.git")
	require.NoError(t, err, "GenerateLoopWorkflow failed")

	require.NotNil(t, wf.Loop, "the loop spec is set")
	assert.Empty(t, wf.Loop.Slug, "the empty slug is carried as-is")
	assert.Equal(t, []string{"run gofmt"}, wf.Loop.Steps, "the steps are carried in the loop spec")

	workflowYAML, err := wf.Render()
	require.NoError(t, err, "Render failed")
	assert.False(t, strings.Contains(workflowYAML, "--slug"), "the rendered workflow omits the empty slug")
}
