package loop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertResolved asserts the returned resolution carries the expected branch
// slug and steps.
func assertResolved(t *testing.T, result *Result, slug string, steps []string) {
	t.Helper()
	require.NotNil(t, result, "the resolution result is returned")
	assert.Equal(t, slug, result.Slug, "the resolved branch slug")
	assert.Equal(t, steps, result.Steps, "the resolved steps")
}

func TestRunPassedStepsReplaceConfigSteps(t *testing.T) {
	steps := []string{"write code", "run tests"}
	client := &mockLoopConfigClient{loops: map[string][]string{
		"fmt": {"run gofmt"},
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, 10)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	assert.True(t, client.called, "the loop config client is consulted to look up the entry when a slug is passed")
	assert.Equal(t, "fmt", client.slug, "the loop config client is called with the passed slug")
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
	assert.True(t, prompt.called, "the prompt builder is called with the passed steps")
	assert.Equal(t, steps, prompt.steps)
}

func TestRunRequiresConfigEntryWhenSlugPassedWithSteps(t *testing.T) {
	steps := []string{"run tests"}
	client := &mockLoopConfigClient{loops: map[string][]string{
		"fmt": {"run gofmt"},
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("missing", steps, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when no loop config matches the slug")
	assert.EqualError(t, err, "loop config not found: missing")
	assert.False(t, prompt.called, "the prompt builder is not called when no loop config matches the slug")
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
	assert.Zero(t, ai.calls, "the AI is not invoked when the resolution fails")
	assert.Zero(t, report.reads, "the report is not read when the resolution fails")
}

func TestRunProposesSlugForPassedSteps(t *testing.T) {
	steps := []string{"write code", "run tests"}
	client := &mockLoopConfigClient{}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "fmt"}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("", steps, 10)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	assert.False(t, client.called, "the loop config client is not consulted when steps are passed without a slug")
	assert.True(t, proposer.called, "the slug proposer is asked for a slug when none is given")
	assert.Equal(t, steps, proposer.steps)
	assert.True(t, prompt.called, "the prompt builder is called with the passed steps")
	assert.Equal(t, steps, prompt.steps)
}

func TestRunPropagatesSlugProposalError(t *testing.T) {
	steps := []string{"write code"}
	proposeErr := errors.New("no usable slug proposed")
	client := &mockLoopConfigClient{}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{err: proposeErr}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("", steps, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when slug proposal fails")
	assert.Equal(t, proposeErr, err)
	assert.False(t, client.called, "the loop config client is not consulted when steps are passed without a slug")
	assert.False(t, prompt.called, "the prompt builder is not called when slug proposal fails")
	assert.Zero(t, ai.calls, "the AI is not invoked when the resolution fails")
	assert.Zero(t, report.reads, "the report is not read when the resolution fails")
}

func TestRunUsesMatchingLoopConfigSteps(t *testing.T) {
	steps := []string{"run gofmt", "run go vet"}
	client := &mockLoopConfigClient{loops: map[string][]string{
		"fmt": steps,
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", nil, 10)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
	assert.True(t, prompt.called, "the prompt builder is called with the matching entry's steps")
	assert.Equal(t, steps, prompt.steps)
}

func TestRunReturnsLoopConfigNotFoundWithoutBuildingPrompt(t *testing.T) {
	client := &mockLoopConfigClient{loops: map[string][]string{
		"fmt": {"run gofmt"},
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("missing", nil, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when no loop config matches")
	assert.EqualError(t, err, "loop config not found: missing")
	assert.False(t, prompt.called, "the prompt builder is not called when no loop config matches")
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
	assert.Zero(t, ai.calls, "the AI is not invoked when the resolution fails")
	assert.Zero(t, report.reads, "the report is not read when the resolution fails")
}

func TestRunPropagatesLoopConfigLookupError(t *testing.T) {
	lookupErr := errors.New("loop config lookup boom")
	client := &mockLoopConfigClient{err: lookupErr}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", nil, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the loop config lookup fails")
	assert.Equal(t, lookupErr, err)
	assert.False(t, prompt.called, "the prompt builder is not called when the loop config lookup fails")
	assert.Zero(t, ai.calls, "the AI is not invoked when the resolution fails")
	assert.Zero(t, report.reads, "the report is not read when the resolution fails")
}

func TestRunPropagatesPromptBuildError(t *testing.T) {
	client := &mockLoopConfigClient{loops: map[string][]string{
		"fmt": {"run gofmt"},
	}}
	promptErr := errors.New("prompt build boom")
	prompt := &mockPromptBuilder{err: promptErr}
	proposer := &mockSlugProposer{}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}
	git := &mockGitClient{}

	result, err := NewCmd(client, prompt, proposer, ai, report, git, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", nil, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the prompt fails to build")
	assert.Equal(t, promptErr, err)
	assert.True(t, prompt.called)
	assert.Equal(t, 1, git.switchCalls, "the loop branch is switched to before the prompt is built")
	assert.Zero(t, ai.calls, "the AI is not invoked when the prompt fails to build")
	assert.Zero(t, report.reads, "the report is not read when the prompt fails to build")
}

// TestRunSwitchesToLoopBranchBeforeIteration asserts the resolved slug reaches
// the git client as a branch switch before any agent pass runs, mirroring how
// `ralph run` switches to the project branch before iterating.
func TestRunSwitchesToLoopBranchBeforeIteration(t *testing.T) {
	steps := []string{"run gofmt"}
	client := &mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}
	git := &mockGitClient{}

	result, err := NewCmd(client, prompt, proposer, ai, report, git, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, 10)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	assert.Equal(t, 1, git.switchCalls, "the loop branch is switched to once before the iterations run")
	assert.Equal(t, []string{"fmt"}, git.switchSlugs, "the loop branch switch receives the resolved slug")
	assert.Zero(t, git.calls, "no iteration is committed before the agent runs")
}

// TestRunPropagatesSwitchToLoopBranchError asserts a loop branch switch failure
// aborts the run before the agent runs and is returned unchanged.
func TestRunPropagatesSwitchToLoopBranchError(t *testing.T) {
	steps := []string{"run gofmt"}
	client := &mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}
	switchErr := errors.New("failed to checkout review branch: boom")
	git := &mockGitClient{switchErr: switchErr}

	result, err := NewCmd(client, prompt, proposer, ai, report, git, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the loop branch switch fails")
	assert.Equal(t, switchErr, err, "the loop branch switch error is returned unchanged")
	assert.False(t, prompt.called, "the prompt is not built when the loop branch switch fails")
	assert.Zero(t, ai.calls, "the AI is not invoked when the loop branch switch fails")
	assert.Zero(t, report.reads, "the report is not read when the loop branch switch fails")
}

func TestRunWithNoSlugAndNoStepsResolvesEmptyAndBuildsEmptyPrompt(t *testing.T) {
	client := &mockLoopConfigClient{}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("", nil, 10)

	require.NoError(t, err)
	assertResolved(t, result, "", nil)
	assert.False(t, client.called, "the loop config client is not consulted when no slug is given")
	assert.False(t, proposer.called, "the slug proposer is not called when there are no steps")
	assert.True(t, prompt.called, "the prompt builder is called with no steps")
	assert.Empty(t, prompt.steps)
	assert.Equal(t, 1, ai.calls, "the AI is invoked once before the nothing-to-do report stops the loop")
	assert.Equal(t, 1, report.reads, "the report is read once before the nothing-to-do report stops the loop")
}

// TestRunDelegatesPullRequestOpening asserts the pull request for the loop
// branch is opened exactly once after the loop ends, whatever the loop's
// outcome, and receives the resolved slug.
func TestRunDelegatesPullRequestOpening(t *testing.T) {
	tests := []struct {
		name     string
		slug     string
		steps    []string
		reports  []string
		max      int
		proposed string
		wantSlug string
	}{
		{
			name:     "opens the pull request when nothing to do first",
			slug:     "fmt",
			steps:    []string{"run gofmt"},
			reports:  nothingToDoReports(),
			max:      10,
			wantSlug: "fmt",
		},
		{
			name:     "opens the pull request after committing work then nothing to do",
			slug:     "fmt",
			steps:    []string{"run gofmt"},
			reports:  []string{"did the work", "NOTHING_TO_DO"},
			max:      10,
			wantSlug: "fmt",
		},
		{
			name:     "opens the pull request after running until max iterations",
			slug:     "fmt",
			steps:    []string{"run gofmt"},
			reports:  []string{"did the work"},
			max:      3,
			wantSlug: "fmt",
		},
		{
			name:     "opens the pull request for the proposed slug",
			slug:     "",
			steps:    []string{"run gofmt"},
			reports:  nothingToDoReports(),
			max:      10,
			proposed: "fmt",
			wantSlug: "fmt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}}
			prompt := &mockPromptBuilder{}
			proposer := &mockSlugProposer{slug: tt.proposed}
			ai := &mockAIClient{}
			report := &mockReportReader{reports: tt.reports}
			pr := &mockPullRequestOpener{}

			result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, pr, envNotInWorkflow()).Run(tt.slug, tt.steps, tt.max)

			require.NoError(t, err)
			assertResolved(t, result, tt.wantSlug, tt.steps)
			assert.Equal(t, 1, pr.calls, "the pull request is opened exactly once after the loop ends")
			assert.Equal(t, []string{tt.wantSlug}, pr.slugs, "the pull request is opened for the resolved slug")
		})
	}
}

// TestRunPropagatesPullRequestOpenError asserts a pull request open failure
// after the loop ends aborts the run and is returned unchanged.
func TestRunPropagatesPullRequestOpenError(t *testing.T) {
	steps := []string{"run gofmt"}
	client := &mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: nothingToDoReports()}
	prErr := errors.New("failed to open loop pull request: boom")
	pr := &mockPullRequestOpener{err: prErr}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, pr, envNotInWorkflow()).Run("fmt", steps, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when opening the pull request fails")
	assert.Equal(t, prErr, err, "the pull request open error is returned unchanged")
	assert.Equal(t, 1, pr.calls, "the pull request open is attempted once after the loop ends")
}

// TestRunLoopStatsPrintedOnSuccess asserts the accumulated AI token usage and
// cost statistics are printed when the loop succeeds inside a workflow
// container, matching `ralph run`.
func TestRunLoopStatsPrintedOnSuccess(t *testing.T) {
	client := &mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}}
	ai := &mockAIClient{}
	cmd := NewCmd(client, &mockPromptBuilder{}, &mockSlugProposer{}, ai, &mockReportReader{reports: nothingToDoReports()}, &mockGitClient{}, &mockPullRequestOpener{}, envInWorkflow())

	result, err := cmd.Run("fmt", nil, 10)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", []string{"run gofmt"})
	require.True(t, ai.statsPrinted, "the stats are printed when the loop succeeds in a workflow")
}

// TestRunLoopStatsPrintedOnFailure asserts the accumulated AI token usage and
// cost statistics are printed before the error is surfaced when the loop fails
// inside a workflow container.
func TestRunLoopStatsPrintedOnFailure(t *testing.T) {
	aiErr := errors.New("opencode execution failed: boom")
	client := &mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}}
	ai := &mockAIClient{err: aiErr}
	cmd := NewCmd(client, &mockPromptBuilder{}, &mockSlugProposer{}, ai, &mockReportReader{reports: nothingToDoReports()}, &mockGitClient{}, &mockPullRequestOpener{}, envInWorkflow())

	result, err := cmd.Run("fmt", nil, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the loop fails")
	assert.Equal(t, aiErr, err, "the AI error is returned unchanged")
	require.True(t, ai.statsPrinted, "the stats are printed before the loop failure is surfaced")
}

// TestRunLoopStatsNotPrintedWhenNotInWorkflow asserts no token usage or cost
// statistics are printed when the loop runs locally with --local.
func TestRunLoopStatsNotPrintedWhenNotInWorkflow(t *testing.T) {
	client := &mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}}
	ai := &mockAIClient{}
	cmd := NewCmd(client, &mockPromptBuilder{}, &mockSlugProposer{}, ai, &mockReportReader{reports: nothingToDoReports()}, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow())

	result, err := cmd.Run("fmt", nil, 10)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", []string{"run gofmt"})
	require.False(t, ai.statsPrinted, "no stats are printed when the loop runs outside a workflow")
}

// TestRunResolvedInWorktreeSkipsBranchSwitch asserts worktree execution never
// switches branches: the git client is not asked to switch to the loop branch,
// yet the loop runs in-process and commits its iterations.
func TestRunResolvedInWorktreeSkipsBranchSwitch(t *testing.T) {
	ai := &mockAIClient{}
	git := &mockGitClient{}
	pr := &mockPullRequestOpener{}
	cmd := NewCmd(&mockLoopConfigClient{}, &mockPromptBuilder{}, &mockSlugProposer{}, ai, &mockReportReader{reports: nothingToDoReports()}, git, pr, envNotInWorkflow())

	err := cmd.RunResolvedInWorktree(&Result{Slug: "fmt", Steps: []string{"run gofmt"}}, 10)

	require.NoError(t, err)
	assert.Zero(t, git.switchCalls, "worktree execution must not switch branches in the current checkout")
	assert.Equal(t, 1, ai.calls, "the AI is invoked once before the nothing-to-do report stops the loop")
	assert.Zero(t, git.calls, "a nothing-to-do iteration is not committed")
	assert.Equal(t, 1, pr.calls, "the pull request is opened after the loop ends")
}

// TestRunResolvedInWorktreeRunsFullLoop asserts worktree execution runs the
// same iteration loop and opens the loop branch's pull request, committing each
// work iteration without ever switching branches.
func TestRunResolvedInWorktreeRunsFullLoop(t *testing.T) {
	ai := &mockAIClient{}
	git := &mockGitClient{}
	pr := &mockPullRequestOpener{}
	cmd := NewCmd(&mockLoopConfigClient{}, &mockPromptBuilder{}, &mockSlugProposer{}, ai, &mockReportReader{reports: []string{"did the work"}}, git, pr, envNotInWorkflow())

	err := cmd.RunResolvedInWorktree(&Result{Slug: "fmt", Steps: []string{"run gofmt"}}, 1)

	require.NoError(t, err)
	assert.Zero(t, git.switchCalls, "worktree execution must not switch branches")
	assert.Equal(t, 1, ai.calls, "the AI runs exactly max iterations when the report never says nothing to do")
	assert.Equal(t, 1, git.calls, "the work iteration is committed")
	assert.Equal(t, []string{"fmt"}, git.slugs, "the commit records the resolved slug")
	assert.Equal(t, 1, pr.calls, "the pull request is opened after the loop ends")
}

// TestRunResolvedInWorktreePropagatesAIError asserts an AI failure inside the
// worktree aborts the loop and is returned unchanged.
func TestRunResolvedInWorktreePropagatesAIError(t *testing.T) {
	aiErr := errors.New("opencode execution failed: boom")
	ai := &mockAIClient{err: aiErr}
	git := &mockGitClient{}
	pr := &mockPullRequestOpener{}
	cmd := NewCmd(&mockLoopConfigClient{}, &mockPromptBuilder{}, &mockSlugProposer{}, ai, &mockReportReader{reports: nothingToDoReports()}, git, pr, envNotInWorkflow())

	err := cmd.RunResolvedInWorktree(&Result{Slug: "fmt", Steps: []string{"run gofmt"}}, 10)

	require.Error(t, err)
	assert.Equal(t, aiErr, err, "the AI error is returned unchanged")
	assert.Zero(t, git.switchCalls, "no branch switch happens before the failure")
	assert.Zero(t, pr.calls, "the pull request is not opened when the loop fails")
}
