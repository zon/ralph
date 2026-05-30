package services

import "github.com/zon/ralph/internal/config"

type MockClient struct {
	RunBeforeFunc   func(cfg *config.RalphConfig) error
	StartFunc       func(cfg *config.RalphConfig) (*Manager, error)
	StopFunc        func(svc *Manager)
	RemoveLogsFunc  func(cfg *config.RalphConfig)
	startCount      int
	stopCount       int
	removeLogsCount int
}

func (m *MockClient) RunBeforeCommands(cfg *config.RalphConfig) error {
	if m.RunBeforeFunc != nil {
		return m.RunBeforeFunc(cfg)
	}
	return nil
}

func (m *MockClient) Start(cfg *config.RalphConfig) (*Manager, error) {
	m.startCount++
	if m.StartFunc != nil {
		return m.StartFunc(cfg)
	}
	return &Manager{}, nil
}

func (m *MockClient) Stop(svc *Manager) {
	m.stopCount++
	if m.StopFunc != nil {
		m.StopFunc(svc)
	}
}

func (m *MockClient) RemoveLogs(cfg *config.RalphConfig) {
	m.removeLogsCount++
	if m.RemoveLogsFunc != nil {
		m.RemoveLogsFunc(cfg)
	}
}
