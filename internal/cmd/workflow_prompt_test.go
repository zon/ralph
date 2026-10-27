package cmd

import (
	"errors"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	orchestrationPrompt "github.com/zon/ralph/internal/orchestration/prompt"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
)

// fakeWorkflowPromptWorkspace records the workspace flags and returns an
// injected error when set, so tests never prepare a real container workspace.
type fakeWorkflowPromptWorkspace struct {
	flags  wksp.WorkspaceFlags
	err    error
	called bool
}

func (f *fakeWorkflowPromptWorkspace) Setup(flags wksp.WorkspaceFlags) error {
	f.called = true
	f.flags = flags
	return f.err
}

// fakeWorkflowPromptRunner records the prompt invocation and returns an
// injected error when set, so tests never run a real prompt flow.
type fakeWorkflowPromptRunner struct {
	flags  orchestrationPrompt.WorkflowPromptFlags
	err    error
	called bool
}

func (f *fakeWorkflowPromptRunner) Run(flags orchestrationPrompt.WorkflowPromptFlags) error {
	f.called = true
	f.flags = flags
	return f.err
}

// TestWorkflowPromptCmdParsing covers the `ralph workflow prompt` command
// surface that runs inside the workflow container.
func TestWorkflowPromptCmdParsing(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{
		"workflow", "prompt",
		"--repo", "owner/repo",
		"--clone", "main",
		"--branch", "my-feature",
		"--base", "release",
		"--prompt", "Add the endpoint.",
		"--model", "gpt-4",
		"--variant", "high",
		"--agent", "build",
		"--debug", "feature",
		"--no-services",
		"--verbose",
	})
	require.NoError(t, err)

	assert.Equal(t, "owner/repo", cmd.Workflow.Prompt.Repo)
	assert.Equal(t, "main", cmd.Workflow.Prompt.CloneBranch)
	assert.Equal(t, "my-feature", cmd.Workflow.Prompt.Branch)
	assert.Equal(t, "release", cmd.Workflow.Prompt.Base)
	assert.Equal(t, "Add the endpoint.", cmd.Workflow.Prompt.Prompt)
	assert.Equal(t, "gpt-4", cmd.Workflow.Prompt.Model)
	assert.Equal(t, "high", cmd.Workflow.Prompt.Variant)
	assert.Equal(t, "build", cmd.Workflow.Prompt.Agent)
	assert.Equal(t, "feature", cmd.Workflow.Prompt.Debug)
	assert.True(t, cmd.Workflow.Prompt.NoServices)
	assert.True(t, cmd.Workflow.Prompt.Verbose)
}

// TestWorkflowPromptCmdDefaults asserts the container-side prompt command
// defaults match the other workflow subcommands.
func TestWorkflowPromptCmdDefaults(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"workflow", "prompt", "--repo", "owner/repo"})
	require.NoError(t, err)
	assert.Equal(t, "ralph-zon[bot]", cmd.Workflow.Prompt.BotName)
	assert.Equal(t, "ralph-zon[bot]@users.noreply.github.com", cmd.Workflow.Prompt.BotEmail)
	assert.False(t, cmd.Workflow.Prompt.NoServices)
}

// TestWorkflowPromptCmdHelpText asserts the workflow prompt subcommand appears
// in the workflow group help.
func TestWorkflowPromptCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "--help"})
	assert.Contains(t, output, "Invoked by workflows to execute a prompt")
}

// TestWorkflowPromptCmdMissingPromptAbortsBeforeWorkspace asserts a missing or
// empty prompt fails before the workspace is prepared and before the prompt
// flow runs.
func TestWorkflowPromptCmdMissingPromptAbortsBeforeWorkspace(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{name: "missing prompt", prompt: ""},
		{name: "whitespace-only prompt", prompt: "  \n\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := &fakeWorkflowPromptWorkspace{}
			runner := &fakeWorkflowPromptRunner{}
			cmd := &WorkflowPromptCmd{
				Repo:           "owner/repo",
				Branch:         "my-feature",
				Base:           "main",
				Prompt:         tt.prompt,
				workspaceSetup: workspace,
				promptRunner:   runner,
			}

			err := cmd.Run()

			require.ErrorIs(t, err, orchestrationPrompt.ErrMissingPrompt)
			assert.False(t, workspace.called, "the workspace is not prepared for an invalid prompt")
			assert.False(t, runner.called, "the prompt does not run for an invalid prompt")
		})
	}
}

// TestWorkflowPromptCmdRunPreparesWorkspaceAndCarriesOptions asserts the
// container-side command prepares the workspace with the submitted repo, clone
// branch, prompt branch, and bot identity, then runs the prompt with the
// carried prompt, base, model, variant, agent, and service choice.
func TestWorkflowPromptCmdRunPreparesWorkspaceAndCarriesOptions(t *testing.T) {
	workspace := &fakeWorkflowPromptWorkspace{}
	runner := &fakeWorkflowPromptRunner{}
	cmd := &WorkflowPromptCmd{
		Repo:           "owner/repo",
		CloneBranch:    "main",
		Branch:         "my-feature",
		Base:           "release",
		Prompt:         "Add the endpoint.",
		BotName:        "ralph-zon[bot]",
		BotEmail:       "ralph-zon[bot]@users.noreply.github.com",
		Model:          "gpt-4",
		Variant:        "high",
		Agent:          "build",
		Debug:          "feature",
		NoServices:     true,
		Verbose:        true,
		workspaceSetup: workspace,
		promptRunner:   runner,
	}

	err := cmd.Run()

	require.NoError(t, err)
	require.True(t, workspace.called, "the workspace is prepared before the prompt runs")
	assert.Equal(t, "owner/repo", workspace.flags.Repo)
	assert.Equal(t, "main", workspace.flags.CloneBranch)
	assert.Equal(t, "my-feature", workspace.flags.TargetBranch, "the workspace checks out the prompt branch")
	assert.True(t, workspace.flags.CreateBranch, "the workspace creates the prompt branch when missing")
	assert.Equal(t, "ralph-zon[bot]", workspace.flags.BotName)
	assert.Equal(t, "ralph-zon[bot]@users.noreply.github.com", workspace.flags.BotEmail)

	require.True(t, runner.called, "the prompt runs after the workspace is prepared")
	assert.Equal(t, "Add the endpoint.", runner.flags.Prompt)
	assert.Equal(t, "my-feature", runner.flags.Branch)
	assert.Equal(t, "release", runner.flags.BaseBranch)
	assert.Equal(t, "gpt-4", runner.flags.Model)
	assert.Equal(t, "high", runner.flags.Variant)
	assert.Equal(t, "build", runner.flags.Agent)
	assert.Equal(t, "feature", runner.flags.Debug)
	assert.True(t, runner.flags.NoServices)
	assert.True(t, runner.flags.Verbose)
}

// TestWorkflowPromptCmdWorkspaceFailureAbortsBeforePrompt asserts a workspace
// failure stops the container-side command before the prompt runs.
func TestWorkflowPromptCmdWorkspaceFailureAbortsBeforePrompt(t *testing.T) {
	workspaceErr := errors.New("workspace setup boom")
	workspace := &fakeWorkflowPromptWorkspace{err: workspaceErr}
	runner := &fakeWorkflowPromptRunner{}
	cmd := &WorkflowPromptCmd{
		Repo:           "owner/repo",
		Branch:         "my-feature",
		Base:           "main",
		Prompt:         "Add the endpoint.",
		workspaceSetup: workspace,
		promptRunner:   runner,
	}

	err := cmd.Run()

	require.ErrorIs(t, err, workspaceErr)
	assert.True(t, workspace.called)
	assert.False(t, runner.called, "the prompt does not run when the workspace setup fails")
}

// TestPromptCmdRunInWorkspaceRunsThePreparedFlow asserts the container-side
// flow runs the before commands and the pass in the checkout the workspace
// prepared, without proposing a slug, resolving a remote mode, or switching
// branches again.
func TestPromptCmdRunInWorkspaceRunsThePreparedFlow(t *testing.T) {
	var order []string
	aiClient := &promptAIRecorder{order: &order}
	servicesClient := &promptServicesRecorder{order: &order}
	prClient := &promptPRRecorder{order: &order}
	gitClient := &git.MockClient{
		CurrentBranchFunc: func() (string, error) {
			t.Fatal("the prepared flow must not read the starting branch")
			return "", nil
		},
		SwitchToBranchFunc: func(string) error {
			t.Fatal("the prepared flow must not switch branches")
			return nil
		},
		FetchBranchFunc: func(branch string) error {
			assert.Equal(t, "release", branch)
			promptEvent(&order, "fetch-base")
			return nil
		},
		NeedsMergeFunc: func(string) (bool, error) {
			promptEvent(&order, "needs-merge")
			return false, nil
		},
		HeadCommitFunc: func() (string, error) {
			return "unchanged-tip", nil
		},
	}
	cmd := &PromptCmd{
		config:     &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) { return config.Any(), nil }},
		git:        gitClient,
		services:   servicesClient,
		ai:         aiClient,
		pr:         prClient,
		output:     &promptOutputRecorder{},
		NoServices: true,
	}

	err := cmd.RunInWorkspace("do work", "my-feature", "release")

	require.NoError(t, err)
	require.NotNil(t, cmd.resolved)
	assert.Equal(t, config.ModeLocal, cmd.resolved.Mode)
	assert.Equal(t, "my-feature", cmd.resolved.PromptBranch)
	assert.Equal(t, "release", cmd.resolved.BaseBranch)
	assert.Equal(t, []string{"before", "fetch-base", "needs-merge", "pass", "remove-logs"}, order)
	assert.Equal(t, []string{"do work"}, aiClient.prompts)
	assert.Zero(t, prClient.calls, "an unchanged tip opens no pull request")
}

// TestPromptCmdRunInWorkspaceRejectsEmptyPrompt asserts the prepared entrypoint
// rejects an empty prompt before loading the configuration.
func TestPromptCmdRunInWorkspaceRejectsEmptyPrompt(t *testing.T) {
	loadCalls := 0
	cmd := &PromptCmd{
		config: &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
			loadCalls++
			return config.Any(), nil
		}},
	}

	err := cmd.RunInWorkspace("  ", "my-feature", "main")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
	assert.Zero(t, loadCalls)
}

// TestPromptWorkflowRunnerAdapterCarriesPrompt asserts the in-process adapter
// hands the prompt, prompt branch, and base branch to the prepared flow.
func TestPromptWorkflowRunnerAdapterCarriesPrompt(t *testing.T) {
	ctx := execcontext.NewContext()
	ctx.SetModel("gpt-4")
	ctx.SetVariant("high")
	ctx.SetAgent("build")
	adapter := &promptWorkflowRunnerAdapter{ctx: ctx}

	// A missing prompt proves the adapter reaches the prepared flow's
	// validation rather than doing its own workspace or branch work.
	err := adapter.Run(orchestrationPrompt.WorkflowPromptFlags{Prompt: " "})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}
