package project

type MockClient struct {
	AllPassingFunc func() bool
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
