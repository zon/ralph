package notify

// MockNotifier implements Notifier with a function field for per-test control.
type MockNotifier struct {
	NotifyFn func(title, message, appIcon string) error
}

func (m *MockNotifier) Notify(title, message, appIcon string) error {
	if m.NotifyFn != nil {
		return m.NotifyFn(title, message, appIcon)
	}
	return nil
}

var _ Notifier = (*MockNotifier)(nil)

type MockClient struct {
	ErrorsSlice    []string
	SuccessesSlice []string
	ErrorFunc      func(slug string)
	SuccessFunc    func(slug string)
}

func (m *MockClient) Error(slug string) {
	m.ErrorsSlice = append(m.ErrorsSlice, slug)
	if m.ErrorFunc != nil {
		m.ErrorFunc(slug)
	}
}

func (m *MockClient) Success(slug string) {
	m.SuccessesSlice = append(m.SuccessesSlice, slug)
	if m.SuccessFunc != nil {
		m.SuccessFunc(slug)
	}
}
