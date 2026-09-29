package loop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIterateNonFatalAIFailureCarriesErrorIntoNextPrompt asserts a non-fatal
// AI pass failure does not stop the loop: the next iteration's prompt carries
// the recorded error as the previous attempt's failure, and a successful
// iteration clears it.
func TestIterateNonFatalAIFailureCarriesErrorIntoNextPrompt(t *testing.T) {
	steps := []string{"run gofmt"}
	aiErr := errors.New("opencode execution failed: boom")
	ai := &mockAIClient{errs: []error{aiErr, nil}}
	prompt := &mockPromptBuilder{}
	report := &mockReportReader{reports: []string{"did the work"}}
	git := &mockGitClient{}

	result, err := NewCmd(&mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}, prompt, &mockSlugProposer{slug: "proposed"}, ai, report, git, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, 3)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	require.Len(t, prompt.previousErrors, 3)
	assert.Nil(t, prompt.previousErrors[0], "the first prompt carries no previous error")
	assert.Equal(t, aiErr, prompt.previousErrors[1], "the next prompt carries the recorded error")
	assert.Nil(t, prompt.previousErrors[2], "a successful iteration clears the recorded error")
	assert.Equal(t, 3, ai.calls)
	assert.Equal(t, 2, report.reads, "a failed AI pass stops before the report is read")
	assert.Equal(t, 2, git.calls, "only successful iterations are committed")
}

// TestIterateCarriesOnlyMostRecentFailure asserts consecutive failures carry
// only the immediately preceding failure into the next prompt.
func TestIterateCarriesOnlyMostRecentFailure(t *testing.T) {
	steps := []string{"run gofmt"}
	firstErr := errors.New("first failure")
	secondErr := errors.New("second failure")
	ai := &mockAIClient{errs: []error{firstErr, secondErr, nil}}
	prompt := &mockPromptBuilder{}
	report := &mockReportReader{reports: []string{"did the work"}}

	result, err := NewCmd(&mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}, prompt, &mockSlugProposer{slug: "proposed"}, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, 3)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	require.Len(t, prompt.previousErrors, 3)
	assert.Nil(t, prompt.previousErrors[0])
	assert.Equal(t, firstErr, prompt.previousErrors[1])
	assert.Equal(t, secondErr, prompt.previousErrors[2], "only the most recent failure is carried")
}

// TestIterateFatalAIFailureAborts asserts a fatal AI pass failure stops the
// loop immediately and is returned, so no further prompt runs.
func TestIterateFatalAIFailureAborts(t *testing.T) {
	steps := []string{"run gofmt"}
	aiErr := errors.New("billing limit exceeded")
	ai := &mockAIClient{err: aiErr, isFatalFunc: func(err error) bool { return err == aiErr }}
	prompt := &mockPromptBuilder{}
	report := &mockReportReader{reports: []string{"did the work"}}

	result, err := NewCmd(&mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}, prompt, &mockSlugProposer{slug: "proposed"}, ai, report, &mockGitClient{}, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when a fatal failure stops the loop")
	assert.Equal(t, aiErr, err, "the fatal error is returned unchanged")
	assert.Equal(t, 1, ai.calls, "no further AI pass runs after a fatal failure")
	assert.Zero(t, report.reads, "no report is read after a fatal failure")
}

// TestIterateNonFatalFailuresExhaustTheCap asserts repeated non-fatal failures
// still consume the iteration cap instead of extending the loop.
func TestIterateNonFatalFailuresExhaustTheCap(t *testing.T) {
	steps := []string{"run gofmt"}
	ai := &mockAIClient{err: errors.New("non-fatal failure")}
	prompt := &mockPromptBuilder{}
	report := &mockReportReader{reports: []string{"did the work"}}
	git := &mockGitClient{}

	result, err := NewCmd(&mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}, prompt, &mockSlugProposer{slug: "proposed"}, ai, report, git, &mockPullRequestOpener{}, envNotInWorkflow()).Run("fmt", steps, 3)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	assert.Equal(t, 3, ai.calls, "the AI runs exactly the cap's worth of passes")
	assert.Equal(t, 3, len(prompt.previousErrors), "a prompt is built each iteration")
	assert.Zero(t, git.calls, "a failed iteration is not committed")
	assert.Zero(t, report.reads, "a failed AI pass stops before the report is read")
}
