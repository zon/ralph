package project

type MockClient struct {
	AllPassingFunc          func() bool
	HasChangesFunc           func(*Project) bool
	HasSpecFunc              func(*Project) bool
	HasOrchestrationFunc         func(*Project) bool
	RemoveOrchestrationFunc      func(*Project) error
	RemoveOrchestrationCalled    bool
	NormalizeAndStageCalled      bool
}

func (m *MockClient) Reload(proj *Project) *Project {
	return proj
}

func (m *MockClient) AllRequirementsPassing(_ *Project) bool {
	return m.AllPassingFunc()
}

func (m *MockClient) MaxIterationsError(_ *Project) error {
	return ErrMaxIterationsReached
}

func (m *MockClient) HasChanges(proj *Project) bool {
	if m.HasChangesFunc != nil {
		return m.HasChangesFunc(proj)
	}
	return false
}

func (m *MockClient) NormalizeAndStage(proj *Project) {
	m.NormalizeAndStageCalled = true
}

func (m *MockClient) HasSpec(proj *Project) bool {
	if m.HasSpecFunc != nil {
		return m.HasSpecFunc(proj)
	}
	return false
}

func (m *MockClient) HasOrchestration(proj *Project) bool {
	if m.HasOrchestrationFunc != nil {
		return m.HasOrchestrationFunc(proj)
	}
	return false
}

func (m *MockClient) RemoveOrchestration(proj *Project) error {
	m.RemoveOrchestrationCalled = true
	if m.RemoveOrchestrationFunc != nil {
		return m.RemoveOrchestrationFunc(proj)
	}
	return nil
}
