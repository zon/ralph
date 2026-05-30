package project

type MockClient struct {
	AllPassingFunc func() bool
}

func (m *MockClient) AllRequirementsPassing(_ *Project) bool {
	return m.AllPassingFunc()
}

func (m *MockClient) MaxIterationsError(_ *Project) error {
	return ErrMaxIterationsReached
}
