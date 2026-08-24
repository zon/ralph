package loop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wksp "github.com/zon/ralph/internal/orchestration/workspace"
)

// mockWorkspaceSetupClient records the workspace flags and returns an injected
// error when set.
type mockWorkspaceSetupClient struct {
	flags  wksp.WorkspaceFlags
	err    error
	called bool
}

func (m *mockWorkspaceSetupClient) Setup(flags wksp.WorkspaceFlags) error {
	m.called = true
	m.flags = flags
	return m.err
}

// mockLoopRunnerClient records the loop invocation and returns an injected
// error when set.
type mockLoopRunnerClient struct {
	slug   string
	steps  []string
	max    int
	err    error
	called bool
}

func (m *mockLoopRunnerClient) Run(slug string, steps []string, max int) error {
	m.called = true
	m.slug = slug
	m.steps = steps
	m.max = max
	return m.err
}

// TestWorkflowLoopCmdPreparesWorkspaceAndRunsLoop asserts the container-side
// workflow loop command sets up the workspace first and then runs the loop
// in-process with the submitted slug, steps, and max iterations.
func TestWorkflowLoopCmdPreparesWorkspaceAndRunsLoop(t *testing.T) {
	workspace := &mockWorkspaceSetupClient{}
	runner := &mockLoopRunnerClient{}
	cmd := NewWorkflowLoopCmd(workspace, runner)

	flags := WorkflowLoopFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
		BotName:     "ralph-zon[bot]",
		BotEmail:    "ralph-zon[bot]@users.noreply.github.com",
		Slug:        "fmt",
		Steps:       []string{"run gofmt", "run go vet"},
		Max:         3,
	}
	err := cmd.Run(flags)

	require.NoError(t, err)
	assert.True(t, workspace.called, "the workspace is prepared before the loop runs")
	assert.Equal(t, "owner/repo", workspace.flags.Repo, "the workspace receives the repo")
	assert.Equal(t, "main", workspace.flags.CloneBranch, "the workspace receives the clone branch")
	assert.Equal(t, "ralph-zon[bot]", workspace.flags.BotName, "the workspace receives the bot name")
	assert.Equal(t, "ralph-zon[bot]@users.noreply.github.com", workspace.flags.BotEmail, "the workspace receives the bot email")
	assert.True(t, runner.called, "the loop runs in-process after the workspace is prepared")
	assert.Equal(t, "fmt", runner.slug, "the loop runs with the submitted slug")
	assert.Equal(t, []string{"run gofmt", "run go vet"}, runner.steps, "the loop runs with the submitted steps")
	assert.Equal(t, 3, runner.max, "the loop runs with the submitted max iterations")
}

// TestWorkflowLoopCmdAbortsOnWorkspaceFailure asserts a workspace setup failure
// aborts the container-side command before the loop runs.
func TestWorkflowLoopCmdAbortsOnWorkspaceFailure(t *testing.T) {
	wsErr := errors.New("workspace setup boom")
	workspace := &mockWorkspaceSetupClient{err: wsErr}
	runner := &mockLoopRunnerClient{}
	cmd := NewWorkflowLoopCmd(workspace, runner)

	err := cmd.Run(WorkflowLoopFlags{Slug: "fmt", Max: 10})

	require.Error(t, err)
	assert.Equal(t, wsErr, err, "the workspace setup error is returned unchanged")
	assert.False(t, runner.called, "the loop does not run when the workspace setup fails")
}

// TestWorkflowLoopCmdPropagatesLoopError asserts a loop failure is returned
// unchanged after the workspace was prepared.
func TestWorkflowLoopCmdPropagatesLoopError(t *testing.T) {
	loopErr := errors.New("loop boom")
	workspace := &mockWorkspaceSetupClient{}
	runner := &mockLoopRunnerClient{err: loopErr}
	cmd := NewWorkflowLoopCmd(workspace, runner)

	err := cmd.Run(WorkflowLoopFlags{Slug: "fmt", Max: 10})

	require.Error(t, err)
	assert.Equal(t, loopErr, err, "the loop error is returned unchanged")
	assert.True(t, workspace.called, "the workspace is still prepared before the loop fails")
	assert.True(t, runner.called, "the loop run is attempted")
}
