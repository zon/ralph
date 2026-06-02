package opencode

import (
	"context"
	"io"
)

type MockOC struct {
	RunCommandFunc func(ctx context.Context, model, variant, prompt string, stdoutWriter, stderrWriter io.Writer) error
	RunAgentFunc   func(ctx context.Context, model, variant, prompt string) error
	GetStatsFunc   func() (Stats, error)
	DisplayStatsFunc func() error
}

func (m *MockOC) RunCommand(ctx context.Context, model, variant, prompt string, stdoutWriter, stderrWriter io.Writer) error {
	if m.RunCommandFunc != nil {
		return m.RunCommandFunc(ctx, model, variant, prompt, stdoutWriter, stderrWriter)
	}
	return nil
}

func (m *MockOC) RunAgent(ctx context.Context, model, variant, prompt string) error {
	if m.RunAgentFunc != nil {
		return m.RunAgentFunc(ctx, model, variant, prompt)
	}
	return nil
}

func (m *MockOC) GetStats() (Stats, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc()
	}
	return Stats{}, nil
}

func (m *MockOC) DisplayStats() error {
	if m.DisplayStatsFunc != nil {
		return m.DisplayStatsFunc()
	}
	return nil
}
