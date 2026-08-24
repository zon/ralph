package loop

import (
	"fmt"

	"github.com/zon/ralph/internal/ai"
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

// mockLoopConfigClient serves the loop config's steps for a slug, recording
// the slug it was called with. It returns the injected error when set.
// Otherwise it returns the steps of the matching loop config entry, or an
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

// nothingToDoReports is the report content that stops the loop, mirroring the
// NOTHING_TO_DO value the loop prompt instructs the agent to write.
func nothingToDoReports() []string {
	return []string{"NOTHING_TO_DO"}
}

// mockAIClient records the prompts it ran and returns an injected error when
// set.
type mockAIClient struct {
	prompts []string
	err     error
	calls   int
}

func (m *mockAIClient) RunAgent(prompt string) error {
	m.calls++
	m.prompts = append(m.prompts, prompt)
	return m.err
}

// mockReportReader serves a sequence of report contents, one per read, and
// returns an injected error when set. Reads past the end of the sequence
// return an empty report.
type mockReportReader struct {
	reports []string
	err     error
	reads   int
}

func (m *mockReportReader) ReadReport() (ai.Report, error) {
	m.reads++
	if m.err != nil {
		return ai.Report{}, m.err
	}
	content := ""
	if m.reads <= len(m.reports) {
		content = m.reports[m.reads-1]
	}
	return ai.Report{Content: content}, nil
}
