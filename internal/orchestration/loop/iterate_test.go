package loop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunIterationLoop covers the loop's stop conditions. The loop stops as
// soon as the agent's report says nothing to do. Otherwise it runs all max
// iterations, invoking the AI, reading the report, and committing the
// iteration once per iteration.
func TestRunIterationLoop(t *testing.T) {
	steps := []string{"run gofmt"}
	tests := []struct {
		name            string
		reports         []string
		max             int
		wantAICalls     int
		wantReads       int
		wantGitCalls    int
		wantPromptReuse bool
	}{
		{
			name:         "stops when the report says nothing to do",
			reports:      nothingToDoReports(),
			max:          10,
			wantAICalls:  1,
			wantReads:    1,
			wantGitCalls: 0,
		},
		{
			name:            "runs every iteration up to max when the report never stops",
			reports:         []string{"did the work"},
			max:             3,
			wantAICalls:     3,
			wantReads:       3,
			wantGitCalls:    3,
			wantPromptReuse: true,
		},
		{
			name:         "commits an iteration then stops on a later nothing-to-do report",
			reports:      []string{"did the work", "NOTHING_TO_DO"},
			max:          10,
			wantAICalls:  2,
			wantReads:    2,
			wantGitCalls: 1,
		},
		{
			name:         "runs no iterations when max is zero",
			reports:      []string{"did the work"},
			max:          0,
			wantAICalls:  0,
			wantReads:    0,
			wantGitCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}
			prompt := &mockPromptBuilder{}
			proposer := &mockSlugProposer{slug: "proposed"}
			ai := &mockAIClient{}
			report := &mockReportReader{reports: tt.reports}
			git := &mockGitClient{}

			result, err := NewCmd(client, prompt, proposer, ai, report, git, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, tt.max)

			require.NoError(t, err)
			assertResolved(t, result, "fmt", steps)
			assert.Equal(t, 1, git.switchCalls, "the loop branch is switched to once before the iterations run")
			assert.Equal(t, []string{"fmt"}, git.switchSlugs, "the loop branch switch receives the resolved slug")
			assert.Equal(t, tt.wantAICalls, ai.calls, "the AI is invoked once per iteration until the loop stops")
			assert.Equal(t, tt.wantReads, report.reads, "the report is read once per iteration until the loop stops")
			assert.Equal(t, tt.wantGitCalls, git.calls, "each non-nothing-to-do iteration is committed exactly once")
			assert.Equal(t, tt.wantGitCalls, len(git.slugs), "every commit records the slug it was called with")
			for _, slug := range git.slugs {
				assert.Equal(t, "fmt", slug, "every iteration commits the resolved slug")
			}
			if tt.wantPromptReuse {
				require.Len(t, ai.prompts, tt.max, "one prompt is recorded per iteration")
				for i := 1; i < len(ai.prompts); i++ {
					assert.Equal(t, ai.prompts[0], ai.prompts[i], "every iteration runs the same built prompt")
				}
			}
		})
	}
}

// TestRunPropagatesAIError asserts an AI failure aborts the loop and is
// returned unchanged, without reading the report.
func TestRunPropagatesAIError(t *testing.T) {
	steps := []string{"run gofmt"}
	client := &mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	aiErr := errors.New("opencode execution failed: boom")
	ai := &mockAIClient{err: aiErr}
	report := &mockReportReader{reports: []string{"did the work"}}

	result, err := NewCmd(client, prompt, proposer, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the AI pass fails")
	assert.Equal(t, aiErr, err, "the AI error is returned unchanged")
	assert.Equal(t, 1, ai.calls, "the AI is invoked once before failing")
	assert.Equal(t, 0, report.reads, "the report is not read when the AI pass fails")
}

// TestRunContinuesWhenReportUnreadable asserts an iteration whose agent pass
// leaves report.md missing or unreadable makes no commit and the loop moves on
// to the next iteration without returning an error. The loop is still bounded
// by the iteration cap: an agent pass that never leaves a readable report runs
// every iteration up to max, committing nothing.
func TestRunContinuesWhenReportUnreadable(t *testing.T) {
	steps := []string{"run gofmt"}
	readErr := errors.New("failed to read report.md: boom")
	tests := []struct {
		name         string
		reports      []string
		alwaysErr    error
		readErrs     map[int]error
		max          int
		wantAICalls  int
		wantReads    int
		wantGitCalls int
	}{
		{
			name:         "skips the unreadable iteration then commits the next work report",
			reports:      []string{"did the work"},
			readErrs:     map[int]error{1: readErr},
			max:          2,
			wantAICalls:  2,
			wantReads:    2,
			wantGitCalls: 1,
		},
		{
			name:         "commits nothing for an unreadable report and stops on a later nothing-to-do report",
			reports:      []string{"NOTHING_TO_DO"},
			readErrs:     map[int]error{1: readErr},
			max:          5,
			wantAICalls:  2,
			wantReads:    2,
			wantGitCalls: 0,
		},
		{
			name:         "an unreadable report on every iteration runs to the cap without committing",
			reports:      []string{"did the work"},
			alwaysErr:    readErr,
			max:          3,
			wantAICalls:  3,
			wantReads:    3,
			wantGitCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}
			prompt := &mockPromptBuilder{}
			proposer := &mockSlugProposer{slug: "proposed"}
			ai := &mockAIClient{}
			report := &mockReportReader{reports: tt.reports, err: tt.alwaysErr, readErrs: tt.readErrs}
			git := &mockGitClient{}

			result, err := NewCmd(client, prompt, proposer, ai, report, git, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, tt.max)

			require.NoError(t, err, "an unreadable report must not fail the loop")
			assertResolved(t, result, "fmt", steps)
			assert.Equal(t, tt.wantAICalls, ai.calls, "the AI runs once per iteration until the loop stops or the cap is reached")
			assert.Equal(t, tt.wantReads, report.reads, "the report is read once per iteration until the loop stops or the cap is reached")
			assert.Equal(t, tt.wantGitCalls, git.calls, "only iterations whose report is readable and reports work are committed")
			assert.Equal(t, tt.wantGitCalls, len(git.slugs), "every commit records the slug it was called with")
		})
	}
}

// TestRunPropagatesIterationCommitError asserts a commit failure aborts the
// loop after the first non-nothing-to-do report and is returned unchanged.
func TestRunPropagatesIterationCommitError(t *testing.T) {
	steps := []string{"run gofmt"}
	client := &mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	ai := &mockAIClient{}
	commitErr := errors.New("failed to push loop-fmt: boom")
	report := &mockReportReader{reports: []string{"did the work"}}
	git := &mockGitClient{err: commitErr}
	pr := &mockPullRequestOpener{}

	result, err := NewCmd(client, prompt, proposer, ai, report, git, pr, envNotInWorkflow()).Run("fmt", steps, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the iteration commit fails")
	assert.Equal(t, commitErr, err, "the iteration commit error is returned unchanged")
	assert.Equal(t, 1, ai.calls, "the AI is invoked once before the commit fails")
	assert.Equal(t, 1, report.reads, "the report is read once before the commit fails")
	assert.Equal(t, 1, git.calls, "the iteration is committed once before the commit fails")
	assert.Equal(t, []string{"fmt"}, git.slugs, "the commit receives the resolved slug")
	assert.Zero(t, pr.calls, "the pull request is not opened when the iteration commit fails")
}

// TestRunCommitsToProposedSlug asserts the slug resolved by the proposer, not
// the raw input slug, is what reaches the git client.
func TestRunCommitsToProposedSlug(t *testing.T) {
	steps := []string{"run gofmt"}
	client := &mockLoopConfigClient{}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "fmt"}
	ai := &mockAIClient{}
	report := &mockReportReader{reports: []string{"did the work"}}
	git := &mockGitClient{}

	result, err := NewCmd(client, prompt, proposer, ai, report, git, &mockPullRequestOpener{}, envNotInWorkflow()).Run("", steps, 1)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	assert.True(t, proposer.called, "the slug proposer is asked for a slug when none is given")
	assert.Equal(t, 1, ai.calls, "the AI is invoked once before the work report is committed")
	assert.Equal(t, 1, report.reads, "the report is read once before the work report is committed")
	assert.Equal(t, 1, git.calls, "the iteration is committed once for the work report")
	assert.Equal(t, []string{"fmt"}, git.slugs, "the commit receives the proposed slug, not the empty input slug")
}
