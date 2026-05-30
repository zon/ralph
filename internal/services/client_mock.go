package services

import "github.com/zon/ralph/internal/config"

type MockClient struct {
	RunBeforeFunc func(cfg *config.RalphConfig) error
}

func (m *MockClient) RunBeforeCommands(cfg *config.RalphConfig) error {
	if m.RunBeforeFunc != nil {
		return m.RunBeforeFunc(cfg)
	}
	return nil
}
