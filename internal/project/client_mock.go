package project

type MockClient struct {
	AllPassingFunc          func() bool
	HasChangesFunc          func(*Project) bool
	NormalizeAndStageCalled bool
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
