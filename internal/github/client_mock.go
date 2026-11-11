package github

import (
	"context"
)

type MockGH struct {
	IsReadyFn              func() bool
	FindExistingPRFn       func(head string) (string, error)
	CreatePRFn             func(title, body, base, head string) (string, error)
	ListOpenPullRequestsFn func() ([]PullRequest, error)
	ListCollaboratorsFn    func(ctx context.Context, owner, repo string) ([]string, error)
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

func (m *MockGH) ListOpenPullRequests() ([]PullRequest, error) {
	if m.ListOpenPullRequestsFn != nil {
		return m.ListOpenPullRequestsFn()
	}
	return nil, nil
}

func (m *MockGH) ListCollaborators(ctx context.Context, owner, repo string) ([]string, error) {
	if m.ListCollaboratorsFn != nil {
		return m.ListCollaboratorsFn(ctx, owner, repo)
	}
	return nil, nil
}
