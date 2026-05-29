package github

import "github.com/zon/ralph/internal/project"

type MockRunAdapter struct {
	CreatePRFunc        func(*project.Project) error
	CreatePRCalled      bool
	CreatePRReturnedNil bool
}

func (m *MockRunAdapter) CreatePR(proj *project.Project) error {
	m.CreatePRCalled = true
	if m.CreatePRFunc != nil {
		err := m.CreatePRFunc(proj)
		if err == nil {
			m.CreatePRReturnedNil = true
		}
		return err
	}
	return nil
}
