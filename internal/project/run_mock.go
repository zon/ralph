package project

type MockRunAdapter struct {
	AllPassingFunc func() bool
}

func (m *MockRunAdapter) AllRequirementsPassing(_ *Project) bool {
	return m.AllPassingFunc()
}

func (m *MockRunAdapter) MaxIterationsError(_ *Project) error {
	return ErrMaxIterationsReached
}
