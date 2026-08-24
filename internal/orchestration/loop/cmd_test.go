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

func configWithLoops(loops ...config.LoopConfig) *config.RalphConfig {
	cfg := config.Any()
	cfg.Loops = loops
	return cfg
}

func TestRunUsesMatchingLoopConfigSteps(t *testing.T) {
	steps := []string{"run gofmt", "run go vet"}
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		return configWithLoops(config.LoopConfig{Slug: "fmt", Steps: steps}), nil
	}}
	prompt := &mockPromptBuilder{}

	err := NewCmd(loader, prompt).Run("fmt", nil)

	require.NoError(t, err)
	assert.True(t, prompt.called, "the prompt builder is called with the matching entry's steps")
	assert.Equal(t, steps, prompt.steps)
}

func TestRunReturnsLoopConfigNotFoundWithoutBuildingPrompt(t *testing.T) {
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		return configWithLoops(config.LoopConfig{Slug: "fmt", Steps: []string{"run gofmt"}}), nil
	}}
	prompt := &mockPromptBuilder{}

	err := NewCmd(loader, prompt).Run("missing", nil)

	require.Error(t, err)
	assert.EqualError(t, err, "loop config not found: missing")
	assert.False(t, prompt.called, "the prompt builder is not called when no loop config matches")
}

func TestRunUsesPassedStepsWithoutLoadingConfig(t *testing.T) {
	steps := []string{"write code", "run tests"}
	loaded := false
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		loaded = true
		return config.Any(), nil
	}}
	prompt := &mockPromptBuilder{}

	err := NewCmd(loader, prompt).Run("", steps)

	require.NoError(t, err)
	assert.False(t, loaded, "the config is not loaded when steps are passed without a slug")
	assert.True(t, prompt.called)
	assert.Equal(t, steps, prompt.steps)
}

func TestRunPropagatesConfigLoadError(t *testing.T) {
	loadErr := errors.New("config load boom")
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		return nil, loadErr
	}}
	prompt := &mockPromptBuilder{}

	err := NewCmd(loader, prompt).Run("fmt", nil)

	require.Error(t, err)
	assert.Equal(t, loadErr, err)
	assert.False(t, prompt.called, "the prompt builder is not called when the config fails to load")
}

func TestRunPropagatesPromptBuildError(t *testing.T) {
	loader := &config.MockLoader{LoadFn: func() (*config.RalphConfig, error) {
		return configWithLoops(config.LoopConfig{Slug: "fmt", Steps: []string{"run gofmt"}}), nil
	}}
	promptErr := errors.New("prompt build boom")
	prompt := &mockPromptBuilder{err: promptErr}

	err := NewCmd(loader, prompt).Run("fmt", nil)

	require.Error(t, err)
	assert.Equal(t, promptErr, err)
	assert.True(t, prompt.called)
}
