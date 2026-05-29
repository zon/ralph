package notify

type MockRunAdapter struct {
	ErrorsSlice    []string
	SuccessesSlice []string
	ErrorFunc      func(slug string)
	SuccessFunc    func(slug string)
}

func (m *MockRunAdapter) Error(slug string) {
	m.ErrorsSlice = append(m.ErrorsSlice, slug)
	if m.ErrorFunc != nil {
		m.ErrorFunc(slug)
	}
}

func (m *MockRunAdapter) Success(slug string) {
	m.SuccessesSlice = append(m.SuccessesSlice, slug)
	if m.SuccessFunc != nil {
		m.SuccessFunc(slug)
	}
}
