package prompt

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wksp "github.com/zon/ralph/internal/orchestration/workspace"
)

type fakeWorkspaceSetupClient struct {
	flags  wksp.WorkspaceFlags
	err    error
	called bool
}

func (f *fakeWorkspaceSetupClient) Setup(flags wksp.WorkspaceFlags) error {
	f.called = true
	f.flags = flags
	return f.err
}

type fakeRunnerClient struct {
	flags  WorkflowPromptFlags
	err    error
	called bool
}

func (f *fakeRunnerClient) Run(flags WorkflowPromptFlags) error {
	f.called = true
	f.flags = flags
	return f.err
}

func promptFlags() WorkflowPromptFlags {
	return WorkflowPromptFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
		Branch:      "my-feature",
		BaseBranch:  "release",
		Prompt:      "Add the endpoint.",
		BotName:     "ralph-zon[bot]",
		BotEmail:    "ralph-zon[bot]@users.noreply.github.com",
		Model:       "gpt-4",
		Variant:     "high",
		Agent:       "build",
		Debug:       "feature",
		NoServices:  true,
		Verbose:     true,
	}
}

// TestWorkflowPromptCmdRejectsMissingPromptBeforeWorkspace covers a missing or
// whitespace-only prompt failing before the workspace is prepared.
func TestWorkflowPromptCmdRejectsMissingPromptBeforeWorkspace(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{name: "missing prompt", prompt: ""},
		{name: "whitespace-only prompt", prompt: " \n\t "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := promptFlags()
			flags.Prompt = tt.prompt
			workspace := &fakeWorkspaceSetupClient{}
			runner := &fakeRunnerClient{}
			cmd := NewWorkflowPromptCmd(workspace, runner)

			err := cmd.Run(flags)

			require.ErrorIs(t, err, ErrMissingPrompt)
			assert.False(t, workspace.called, "the workspace is not prepared for an invalid prompt")
			assert.False(t, runner.called, "the prompt does not run for an invalid prompt")
		})
	}
}

// TestWorkflowPromptCmdPreparesWorkspaceOnPromptBranch asserts the workspace
// clones the repository, checks out the prompt branch, creating it when it does
// not exist, and then runs the prompt with every option the workflow carried.
func TestWorkflowPromptCmdPreparesWorkspaceOnPromptBranch(t *testing.T) {
	workspace := &fakeWorkspaceSetupClient{}
	runner := &fakeRunnerClient{}
	cmd := NewWorkflowPromptCmd(workspace, runner)
	flags := promptFlags()

	require.NoError(t, cmd.Run(flags))

	assert.True(t, workspace.called, "the workspace is prepared before the prompt runs")
	assert.Equal(t, "owner/repo", workspace.flags.Repo)
	assert.Equal(t, "main", workspace.flags.CloneBranch)
	assert.Equal(t, "my-feature", workspace.flags.TargetBranch, "the workspace checks out the prompt branch")
	assert.True(t, workspace.flags.CreateBranch, "the prompt branch is created when it does not exist")
	assert.Equal(t, "ralph-zon[bot]", workspace.flags.BotName)
	assert.Equal(t, "ralph-zon[bot]@users.noreply.github.com", workspace.flags.BotEmail)

	assert.True(t, runner.called, "the prompt runs after the workspace is prepared")
	assert.Equal(t, "Add the endpoint.", runner.flags.Prompt)
	assert.Equal(t, "my-feature", runner.flags.Branch)
	assert.Equal(t, "release", runner.flags.BaseBranch)
	assert.Equal(t, "gpt-4", runner.flags.Model)
	assert.Equal(t, "high", runner.flags.Variant)
	assert.Equal(t, "build", runner.flags.Agent)
	assert.True(t, runner.flags.NoServices)
	assert.Equal(t, "feature", runner.flags.Debug)
	assert.True(t, runner.flags.Verbose)
}

// TestWorkflowPromptCmdWorkspaceFailureAbortsBeforePrompt asserts a workspace
// failure stops the command before the prompt flow starts.
func TestWorkflowPromptCmdWorkspaceFailureAbortsBeforePrompt(t *testing.T) {
	workspaceErr := errors.New("workspace setup boom")
	workspace := &fakeWorkspaceSetupClient{err: workspaceErr}
	runner := &fakeRunnerClient{}
	cmd := NewWorkflowPromptCmd(workspace, runner)

	err := cmd.Run(promptFlags())

	require.ErrorIs(t, err, workspaceErr)
	assert.True(t, workspace.called)
	assert.False(t, runner.called, "the prompt does not run when the workspace setup fails")
}

// TestWorkflowPromptCmdPropagatesPromptFailure asserts a prompt flow failure is
// returned unchanged after the workspace was prepared.
func TestWorkflowPromptCmdPropagatesPromptFailure(t *testing.T) {
	promptErr := errors.New("prompt boom")
	workspace := &fakeWorkspaceSetupClient{}
	runner := &fakeRunnerClient{err: promptErr}
	cmd := NewWorkflowPromptCmd(workspace, runner)

	err := cmd.Run(promptFlags())

	require.ErrorIs(t, err, promptErr)
	assert.True(t, workspace.called)
	assert.True(t, runner.called)
}
