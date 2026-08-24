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

	result, err := NewCmd(client, prompt, proposer).Run("fmt", steps)

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

	result, err := NewCmd(client, prompt, proposer).Run("missing", steps)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when no loop config matches the slug")
	assert.EqualError(t, err, "loop config not found: missing")
	assert.False(t, prompt.called, "the prompt builder is not called when no loop config matches the slug")
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
}

func TestRunProposesSlugForPassedSteps(t *testing.T) {
	steps := []string{"write code", "run tests"}
	client := &mockLoopConfigClient{}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "fmt"}

	result, err := NewCmd(client, prompt, proposer).Run("", steps)

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

	result, err := NewCmd(client, prompt, proposer).Run("", steps)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when slug proposal fails")
	assert.Equal(t, proposeErr, err)
	assert.False(t, client.called, "the loop config client is not consulted when steps are passed without a slug")
	assert.False(t, prompt.called, "the prompt builder is not called when slug proposal fails")
}

func TestRunUsesMatchingLoopConfigSteps(t *testing.T) {
	steps := []string{"run gofmt", "run go vet"}
	client := &mockLoopConfigClient{loops: map[string][]string{
		"fmt": steps,
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}

	result, err := NewCmd(client, prompt, proposer).Run("fmt", nil)

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

	result, err := NewCmd(client, prompt, proposer).Run("missing", nil)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when no loop config matches")
	assert.EqualError(t, err, "loop config not found: missing")
	assert.False(t, prompt.called, "the prompt builder is not called when no loop config matches")
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
}

func TestRunPropagatesLoopConfigLookupError(t *testing.T) {
	lookupErr := errors.New("loop config lookup boom")
	client := &mockLoopConfigClient{err: lookupErr}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{}

	result, err := NewCmd(client, prompt, proposer).Run("fmt", nil)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the loop config lookup fails")
	assert.Equal(t, lookupErr, err)
	assert.False(t, prompt.called, "the prompt builder is not called when the loop config lookup fails")
}

func TestRunPropagatesPromptBuildError(t *testing.T) {
	client := &mockLoopConfigClient{loops: map[string][]string{
		"fmt": {"run gofmt"},
	}}
	promptErr := errors.New("prompt build boom")
	prompt := &mockPromptBuilder{err: promptErr}
	proposer := &mockSlugProposer{}

	result, err := NewCmd(client, prompt, proposer).Run("fmt", nil)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the prompt fails to build")
	assert.Equal(t, promptErr, err)
	assert.True(t, prompt.called)
}

func TestRunWithNoSlugAndNoStepsBuildsPromptWithoutConsultingAI(t *testing.T) {
	client := &mockLoopConfigClient{}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}

	result, err := NewCmd(client, prompt, proposer).Run("", nil)

	require.NoError(t, err)
	assertResolved(t, result, "", nil)
	assert.False(t, client.called, "the loop config client is not consulted when no slug is given")
	assert.False(t, proposer.called, "the slug proposer is not called when there are no steps")
	assert.True(t, prompt.called, "the prompt builder is called with no steps")
	assert.Empty(t, prompt.steps)
}
