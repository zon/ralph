package loop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunIterationLoop covers the loop's stop conditions. The loop stops as
// soon as the agent's report says nothing to do. Otherwise it runs all max
// iterations, invoking the AI and reading the report once per iteration.
func TestRunIterationLoop(t *testing.T) {
	steps := []string{"run gofmt"}
	tests := []struct {
		name            string
		reports         []string
		max             int
		wantAICalls     int
		wantReads       int
		wantPromptReuse bool
	}{
		{
			name:        "stops when the report says nothing to do",
			reports:     nothingToDoReports(),
			max:         10,
			wantAICalls: 1,
			wantReads:   1,
		},
		{
			name:            "runs every iteration up to max when the report never stops",
			reports:         []string{"did the work"},
			max:             3,
			wantAICalls:     3,
			wantReads:       3,
			wantPromptReuse: true,
		},
		{
			name:        "runs no iterations when max is zero",
			reports:     []string{"did the work"},
			max:         0,
			wantAICalls: 0,
			wantReads:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}
			prompt := &mockPromptBuilder{}
			proposer := &mockSlugProposer{slug: "proposed"}
			ai := &mockAIClient{}
			report := &mockReportReader{reports: tt.reports}

			result, err := NewCmd(client, prompt, proposer, ai, report).Run("fmt", steps, tt.max)

			require.NoError(t, err)
			assertResolved(t, result, "fmt", steps)
			assert.Equal(t, tt.wantAICalls, ai.calls, "the AI is invoked once per iteration until the loop stops")
			assert.Equal(t, tt.wantReads, report.reads, "the report is read once per iteration until the loop stops")
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

	result, err := NewCmd(client, prompt, proposer, ai, report).Run("fmt", steps, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the AI pass fails")
	assert.Equal(t, aiErr, err, "the AI error is returned unchanged")
	assert.Equal(t, 1, ai.calls, "the AI is invoked once before failing")
	assert.Equal(t, 0, report.reads, "the report is not read when the AI pass fails")
}

// TestRunPropagatesReportReadError asserts a report read failure aborts the
// loop and is returned unchanged.
func TestRunPropagatesReportReadError(t *testing.T) {
	steps := []string{"run gofmt"}
	client := &mockLoopConfigClient{loops: map[string][]string{"fmt": steps}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}
	ai := &mockAIClient{}
	readErr := errors.New("failed to read report.md: boom")
	report := &mockReportReader{err: readErr}

	result, err := NewCmd(client, prompt, proposer, ai, report).Run("fmt", steps, 10)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the report read fails")
	assert.Equal(t, readErr, err, "the report read error is returned unchanged")
	assert.Equal(t, 1, ai.calls, "the AI is invoked once before the read fails")
	assert.Equal(t, 1, report.reads, "the report is read once before failing")
}
