package skills

type MockClient struct {
	DiscoverFunc   func(branch string) ([]string, error)
	FetchAllFunc   func(branch string, names []string) ([]Skill, error)
	PruneStaleFunc func(root string, fetched []Skill)
	InstallAllFunc func(root string, fetched []Skill) error
}

func (m *MockClient) Discover(branch string) ([]string, error) {
	if m.DiscoverFunc != nil {
		return m.DiscoverFunc(branch)
	}
	return nil, nil
}

func (m *MockClient) FetchAll(branch string, names []string) ([]Skill, error) {
	if m.FetchAllFunc != nil {
		return m.FetchAllFunc(branch, names)
	}
	return nil, nil
}

func (m *MockClient) PruneStale(root string, fetched []Skill) {
	if m.PruneStaleFunc != nil {
		m.PruneStaleFunc(root, fetched)
	}
}

func (m *MockClient) InstallAll(root string, fetched []Skill) error {
	if m.InstallAllFunc != nil {
		return m.InstallAllFunc(root, fetched)
	}
	return nil
}
