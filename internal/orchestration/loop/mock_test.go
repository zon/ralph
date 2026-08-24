package loop

import "fmt"

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

// mockLoopConfigClient serves the loop config's steps for a slug, recording
// the slug it was called with. It returns the injected error when set,
// otherwise the steps of the matching loop config entry, or the not-found
// error when no entry matches the slug.
type mockLoopConfigClient struct {
	loops  map[string][]string
	err    error
	slug   string
	called bool
}

func (m *mockLoopConfigClient) LoopSteps(slug string) ([]string, error) {
	m.called = true
	m.slug = slug
	if m.err != nil {
		return nil, m.err
	}
	if steps, ok := m.loops[slug]; ok {
		return steps, nil
	}
	return nil, fmt.Errorf("loop config not found: %s", slug)
}
