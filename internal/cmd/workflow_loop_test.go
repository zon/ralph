package cmd

import (
	"errors"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	orchestrationLoop "github.com/zon/ralph/internal/orchestration/loop"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
)

// TestWorkflowLoopCmdParsing covers the `ralph workflow loop` command surface
// that runs inside the workflow container: the required repo, the optional
// clone branch, bot identity defaults, and the slug, repeatable --step, and
// --max flags.
func TestWorkflowLoopCmdParsing(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{
		"workflow", "loop",
		"--repo", "owner/repo",
		"--clone-branch", "main",
		"--slug", "fmt",
		"--step", "run gofmt",
		"--step", "run go vet",
		"--max", "3",
	})
	require.NoError(t, err)
	assert.Equal(t, "owner/repo", cmd.Workflow.Loop.Repo)
	assert.Equal(t, "main", cmd.Workflow.Loop.CloneBranch)
	assert.Equal(t, "fmt", cmd.Workflow.Loop.Slug)
	assert.Equal(t, []string{"run gofmt", "run go vet"}, cmd.Workflow.Loop.Steps)
	assert.Equal(t, 3, cmd.Workflow.Loop.Max)
}

// TestWorkflowLoopCmdDefaults asserts the container-side command defaults match
// `ralph workflow run`: bot identity, max iterations, and no slug or steps.
func TestWorkflowLoopCmdDefaults(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"workflow", "loop", "--repo", "owner/repo"})
	require.NoError(t, err)
	assert.Equal(t, "ralph-zon[bot]", cmd.Workflow.Loop.BotName, "the bot name defaults to the app bot")
	assert.Equal(t, "ralph-zon[bot]@users.noreply.github.com", cmd.Workflow.Loop.BotEmail, "the bot email defaults to the app bot email")
	assert.Equal(t, 20, cmd.Workflow.Loop.Max, "the max iterations default to 20")
	assert.Empty(t, cmd.Workflow.Loop.Slug, "the slug defaults to empty")
	assert.Empty(t, cmd.Workflow.Loop.Steps, "the steps default to empty")
}

// TestWorkflowLoopCmdHelpText asserts the workflow loop subcommand appears in
// the workflow group help.
func TestWorkflowLoopCmdHelpText(t *testing.T) {
	output := captureHelpOutput(&Cmd{}, []string{"workflow", "--help"})
	assert.Contains(t, output, "Run a loop via the workflow engine")
}

// fakeWorkspaceSetupClient records the workspace flags and returns an injected
// error when set, so tests never prepare a real container workspace.
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

// fakeLoopRunnerClient records the loop invocation and returns an injected
// error when set, so tests never run a real loop.
type fakeLoopRunnerClient struct {
	slug   string
	steps  []string
	max    int
	err    error
	called bool
}

func (f *fakeLoopRunnerClient) Run(slug string, steps []string, max int) error {
	f.called = true
	f.slug = slug
	f.steps = steps
	f.max = max
	return f.err
}

// TestWorkflowLoopCmdRunPreparesWorkspaceAndRunsLoop asserts the container-side
// command prepares the workspace with the submitted repo, clone branch, and bot
// identity, then runs the loop in-process with the submitted slug, steps, and
// max iterations.
func TestWorkflowLoopCmdRunPreparesWorkspaceAndRunsLoop(t *testing.T) {
	workspace := &fakeWorkspaceSetupClient{}
	runner := &fakeLoopRunnerClient{}
	cmd := &WorkflowLoopCmd{
		Repo:           "owner/repo",
		CloneBranch:    "main",
		BotName:        "ralph-zon[bot]",
		BotEmail:       "ralph-zon[bot]@users.noreply.github.com",
		Slug:           "fmt",
		Steps:          []string{"run gofmt", "run go vet"},
		Max:            3,
		workspaceSetup: workspace,
		loopRunner:     runner,
	}

	err := cmd.Run()

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

// TestWorkflowLoopCmdRunAbortsOnWorkspaceFailure asserts a workspace setup
// failure aborts the container-side command before the loop runs.
func TestWorkflowLoopCmdRunAbortsOnWorkspaceFailure(t *testing.T) {
	wsErr := errors.New("workspace setup boom")
	workspace := &fakeWorkspaceSetupClient{err: wsErr}
	runner := &fakeLoopRunnerClient{}
	cmd := &WorkflowLoopCmd{Repo: "owner/repo", Slug: "fmt", Max: 10, workspaceSetup: workspace, loopRunner: runner}

	err := cmd.Run()

	require.Error(t, err)
	assert.Equal(t, wsErr, err, "the workspace setup error is returned unchanged")
	assert.False(t, runner.called, "the loop does not run when the workspace setup fails")
}

// TestWorkflowLoopCmdRunPropagatesLoopError asserts a loop failure is returned
// unchanged after the workspace was prepared.
func TestWorkflowLoopCmdRunPropagatesLoopError(t *testing.T) {
	loopErr := errors.New("loop boom")
	workspace := &fakeWorkspaceSetupClient{}
	runner := &fakeLoopRunnerClient{err: loopErr}
	cmd := &WorkflowLoopCmd{Repo: "owner/repo", Slug: "fmt", Max: 10, workspaceSetup: workspace, loopRunner: runner}

	err := cmd.Run()

	require.Error(t, err)
	assert.Equal(t, loopErr, err, "the loop error is returned unchanged")
	assert.True(t, workspace.called, "the workspace is still prepared before the loop fails")
	assert.True(t, runner.called, "the loop run is attempted")
}

// inProcessLoopRunner runs the real loop orchestration with injected
// dependencies, so tests exercise the container-side loop body without invoking
// any external tool.
type inProcessLoopRunner struct {
	cfg     orchestrationLoop.LoopConfigClient
	prompt  orchestrationLoop.PromptBuilder
	propose orchestrationLoop.SlugProposer
	ai      orchestrationLoop.AIClient
	report  orchestrationLoop.ReportReader
	git     orchestrationLoop.GitClient
	pr      orchestrationLoop.PullRequestOpener
}

func (r *inProcessLoopRunner) Run(slug string, steps []string, max int) error {
	_, err := orchestrationLoop.NewCmd(r.cfg, r.prompt, r.propose, r.ai, r.report, r.git, r.pr, &SystemEnvClient{}).Run(slug, steps, max)
	return err
}

// TestWorkflowLoopCmdRunRunsLoopBodyIdenticalToLocal asserts the container-side
// command runs the full loop body: slug and step resolution, prompt
// construction, iteration, commit and push to loop-<slug>, and pull request
// opening, exactly like `ralph loop --local`. The injected runner wraps the
// same loop orchestration the --local path wires, fed with the same fakes the
// --local tests use.
func TestWorkflowLoopCmdRunRunsLoopBodyIdenticalToLocal(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
      - run go vet
`)

	proposer := &fakeSlugProposer{slug: "should-not-be-used"}
	ai := &fakeAIClient{}
	report := &fakeReportReader{content: "did the work"}
	git := &fakeGitClient{}
	pr := &fakePullRequestOpener{}
	runner := &inProcessLoopRunner{
		cfg:     &config.Client{},
		prompt:  &loopPromptBuilder{},
		propose: proposer,
		ai:      ai,
		report:  report,
		git:     git,
		pr:      pr,
	}
	cmd := &WorkflowLoopCmd{
		Repo:           "owner/repo",
		CloneBranch:    "main",
		BotName:        "ralph-zon[bot]",
		BotEmail:       "ralph-zon[bot]@users.noreply.github.com",
		Slug:           "fmt",
		Max:            1,
		workspaceSetup: &fakeWorkspaceSetupClient{},
		loopRunner:     runner,
	}

	err := cmd.Run()

	require.NoError(t, err)
	assert.False(t, proposer.called, "the slug proposer is not called when the slug is submitted")
	require.Len(t, ai.prompts, 1, "the loop body runs the built prompt through the AI")
	assert.Contains(t, ai.prompts[0], "run gofmt", "the prompt embeds the resolved steps in order")
	assert.Contains(t, ai.prompts[0], "run go vet", "the prompt embeds the resolved steps in order")
	assert.Equal(t, 1, git.calls, "the work iteration is committed once")
	assert.Equal(t, []string{"fmt"}, git.slugs, "the iteration is committed and pushed to loop-<slug>")
	assert.Equal(t, 1, git.switchCalls, "the loop branch is switched to once before the iterations run")
	assert.Equal(t, 1, pr.calls, "the pull request is opened once after the loop ends")
	assert.Equal(t, []string{"fmt"}, pr.slugs, "the pull request is opened for the resolved slug")
}

// TestWorkflowLoopCmdPrintsStatsInWorkflow asserts the container-side workflow
// loop command prints the accumulated AI token usage and cost statistics after
// the loop succeeds, matching `ralph run` in the workflow container.
func TestWorkflowLoopCmdPrintsStatsInWorkflow(t *testing.T) {
	t.Setenv("RALPH_WORKFLOW_EXECUTION", "true")
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	ai := &fakeAIClient{}
	runner := &inProcessLoopRunner{
		cfg:     &config.Client{},
		prompt:  &loopPromptBuilder{},
		propose: &fakeSlugProposer{slug: "should-not-be-used"},
		ai:      ai,
		report:  &fakeReportReader{content: "NOTHING_TO_DO"},
		git:     &fakeGitClient{},
		pr:      &fakePullRequestOpener{},
	}
	cmd := &WorkflowLoopCmd{
		Repo:           "owner/repo",
		CloneBranch:    "main",
		BotName:        "ralph-zon[bot]",
		BotEmail:       "ralph-zon[bot]@users.noreply.github.com",
		Slug:           "fmt",
		Max:            1,
		workspaceSetup: &fakeWorkspaceSetupClient{},
		loopRunner:     runner,
	}

	err := cmd.Run()

	require.NoError(t, err)
	require.True(t, ai.statsPrinted, "the stats are printed when the loop succeeds in the workflow container")
}

// TestWorkflowLoopCmdPrintsStatsOnFailureInWorkflow asserts the container-side
// workflow loop command prints the accumulated AI token usage and cost
// statistics before the loop failure is surfaced.
func TestWorkflowLoopCmdPrintsStatsOnFailureInWorkflow(t *testing.T) {
	t.Setenv("RALPH_WORKFLOW_EXECUTION", "true")
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	aiErr := errors.New("loop boom")
	ai := &fakeAIClient{err: aiErr}
	runner := &inProcessLoopRunner{
		cfg:     &config.Client{},
		prompt:  &loopPromptBuilder{},
		propose: &fakeSlugProposer{slug: "should-not-be-used"},
		ai:      ai,
		report:  &fakeReportReader{content: "NOTHING_TO_DO"},
		git:     &fakeGitClient{},
		pr:      &fakePullRequestOpener{},
	}
	cmd := &WorkflowLoopCmd{
		Repo:           "owner/repo",
		CloneBranch:    "main",
		BotName:        "ralph-zon[bot]",
		BotEmail:       "ralph-zon[bot]@users.noreply.github.com",
		Slug:           "fmt",
		Max:            1,
		workspaceSetup: &fakeWorkspaceSetupClient{},
		loopRunner:     runner,
	}

	err := cmd.Run()

	require.Error(t, err)
	require.Equal(t, aiErr, err, "the loop error is returned unchanged")
	require.True(t, ai.statsPrinted, "the stats are printed before the loop failure is surfaced")
}
