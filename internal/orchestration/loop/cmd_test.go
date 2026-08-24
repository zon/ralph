package loop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
)

// mockPromptBuilder records the steps it was called with and returns an
// injected error when set.
type mockPromptBuilder struct {
	steps  []string
	err    error
	called bool
}

func (m *mockPromptBuilder) BuildLoopPrompt(steps []string) (string, error) {
	m.called = true
	m.steps = steps
	if m.err != nil {
		return "", m.err
	}
	return "prompt", nil
}

// mockSlugProposer records the steps it was called with and returns an
// injected slug or error.
type mockSlugProposer struct {
	steps  []string
	slug   string
	err    error
	called bool
}

func (m *mockSlugProposer) ProposeSlug(steps []string) (string, error) {
	m.called = true
	m.steps = steps
	if m.err != nil {
		return "", m.err
	}
	return m.slug, nil
}

func configWithLoops(loops ...config.LoopConfig) *config.RalphConfig {
	cfg := config.Any()
	cfg.Loops = loops
	return cfg
}

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
	loaded := false
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		loaded = true
		return configWithLoops(config.LoopConfig{Slug: "fmt", Steps: []string{"run gofmt"}}), nil
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}

	result, err := NewCmd(loader, prompt, proposer).Run("fmt", steps)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	assert.True(t, loaded, "the config is loaded to look up the entry when a slug is passed")
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
	assert.True(t, prompt.called, "the prompt builder is called with the passed steps")
	assert.Equal(t, steps, prompt.steps)
}

func TestRunRequiresConfigEntryWhenSlugPassedWithSteps(t *testing.T) {
	steps := []string{"run tests"}
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		return configWithLoops(config.LoopConfig{Slug: "fmt", Steps: []string{"run gofmt"}}), nil
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}

	result, err := NewCmd(loader, prompt, proposer).Run("missing", steps)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when no loop config matches the slug")
	assert.EqualError(t, err, "loop config not found: missing")
	assert.False(t, prompt.called, "the prompt builder is not called when no loop config matches the slug")
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
}

func TestRunProposesSlugForPassedSteps(t *testing.T) {
	steps := []string{"write code", "run tests"}
	loaded := false
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		loaded = true
		return config.Any(), nil
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "fmt"}

	result, err := NewCmd(loader, prompt, proposer).Run("", steps)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	assert.False(t, loaded, "the config is not loaded when steps are passed without a slug")
	assert.True(t, proposer.called, "the slug proposer is asked for a slug when none is given")
	assert.Equal(t, steps, proposer.steps)
	assert.True(t, prompt.called, "the prompt builder is called with the passed steps")
	assert.Equal(t, steps, prompt.steps)
}

func TestRunPropagatesSlugProposalError(t *testing.T) {
	steps := []string{"write code"}
	proposeErr := errors.New("no usable slug proposed")
	loaded := false
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		loaded = true
		return config.Any(), nil
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{err: proposeErr}

	result, err := NewCmd(loader, prompt, proposer).Run("", steps)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when slug proposal fails")
	assert.Equal(t, proposeErr, err)
	assert.False(t, loaded, "the config is not loaded when steps are passed without a slug")
	assert.False(t, prompt.called, "the prompt builder is not called when slug proposal fails")
}

func TestRunUsesMatchingLoopConfigSteps(t *testing.T) {
	steps := []string{"run gofmt", "run go vet"}
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		return configWithLoops(config.LoopConfig{Slug: "fmt", Steps: steps}), nil
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}

	result, err := NewCmd(loader, prompt, proposer).Run("fmt", nil)

	require.NoError(t, err)
	assertResolved(t, result, "fmt", steps)
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
	assert.True(t, prompt.called, "the prompt builder is called with the matching entry's steps")
	assert.Equal(t, steps, prompt.steps)
}

func TestRunReturnsLoopConfigNotFoundWithoutBuildingPrompt(t *testing.T) {
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		return configWithLoops(config.LoopConfig{Slug: "fmt", Steps: []string{"run gofmt"}}), nil
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{}

	result, err := NewCmd(loader, prompt, proposer).Run("missing", nil)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when no loop config matches")
	assert.EqualError(t, err, "loop config not found: missing")
	assert.False(t, prompt.called, "the prompt builder is not called when no loop config matches")
	assert.False(t, proposer.called, "the slug proposer is not called when a slug is passed")
}

func TestRunPropagatesConfigLoadError(t *testing.T) {
	loadErr := errors.New("config load boom")
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		return nil, loadErr
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{}

	result, err := NewCmd(loader, prompt, proposer).Run("fmt", nil)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the config fails to load")
	assert.Equal(t, loadErr, err)
	assert.False(t, prompt.called, "the prompt builder is not called when the config fails to load")
}

func TestRunPropagatesPromptBuildError(t *testing.T) {
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		return configWithLoops(config.LoopConfig{Slug: "fmt", Steps: []string{"run gofmt"}}), nil
	}}
	promptErr := errors.New("prompt build boom")
	prompt := &mockPromptBuilder{err: promptErr}
	proposer := &mockSlugProposer{}

	result, err := NewCmd(loader, prompt, proposer).Run("fmt", nil)

	require.Error(t, err)
	assert.Nil(t, result, "no resolution is returned when the prompt fails to build")
	assert.Equal(t, promptErr, err)
	assert.True(t, prompt.called)
}

func TestRunWithNoSlugAndNoStepsBuildsPromptWithoutConsultingAI(t *testing.T) {
	loaded := false
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		loaded = true
		return config.Any(), nil
	}}
	prompt := &mockPromptBuilder{}
	proposer := &mockSlugProposer{slug: "proposed"}

	result, err := NewCmd(loader, prompt, proposer).Run("", nil)

	require.NoError(t, err)
	assertResolved(t, result, "", nil)
	assert.False(t, loaded, "the config is not loaded when no slug is given")
	assert.False(t, proposer.called, "the slug proposer is not called when there are no steps")
	assert.True(t, prompt.called, "the prompt builder is called with no steps")
	assert.Empty(t, prompt.steps)
}
