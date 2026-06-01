package github

import "github.com/zon/ralph/internal/project"

type MockGH struct {
	IsReadyFn         func() bool
	FindExistingPRFn  func(head string) (string, error)
	CreatePRFn        func(title, body, base, head string) (string, error)
	GetPRHeadRefOidFn func(pr string) (string, error)
	MergePRFn         func(pr, repo string) error
}

func (m *MockGH) IsReady() bool {
	if m.IsReadyFn != nil {
		return m.IsReadyFn()
	}
	return false
}

func (m *MockGH) FindExistingPR(head string) (string, error) {
	if m.FindExistingPRFn != nil {
		return m.FindExistingPRFn(head)
	}
	return "", nil
}

func (m *MockGH) CreatePR(title, body, base, head string) (string, error) {
	if m.CreatePRFn != nil {
		return m.CreatePRFn(title, body, base, head)
	}
	return "", nil
}

func (m *MockGH) GetPRHeadRefOid(pr string) (string, error) {
	if m.GetPRHeadRefOidFn != nil {
		return m.GetPRHeadRefOidFn(pr)
	}
	return "", nil
}

func (m *MockGH) MergePR(pr, repo string) error {
	if m.MergePRFn != nil {
		return m.MergePRFn(pr, repo)
	}
	return nil
}

type MockClient struct {
	CreatePRFunc func(*project.Project) error
}

func (m *MockClient) CreatePR(proj *project.Project) error {
	if m.CreatePRFunc != nil {
		return m.CreatePRFunc(proj)
	}
	return nil
}
