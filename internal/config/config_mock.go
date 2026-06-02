package config

type MockLoader struct {
	LoadFn func() (*RalphConfig, error)
}

func (m *MockLoader) Load() (*RalphConfig, error) {
	if m.LoadFn != nil {
		return m.LoadFn()
	}
	return Any(), nil
}
