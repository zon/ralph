package github

import (
	"context"

	"github.com/zon/ralph/internal/project"
)

type MockGH struct {
	IsReadyFn           func() bool
	FindExistingPRFn    func(head string) (string, error)
	CreatePRFn          func(title, body, base, head string) (string, error)
	PostCommentFn       func(prNumber int, body string) error
	ListCollaboratorsFn func(ctx context.Context, owner, repo string) ([]string, error)
	RegisterWebhookFn   func(ctx context.Context, owner, repo, webhookURL, secret string) error
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

func (m *MockGH) PostComment(prNumber int, body string) error {
	if m.PostCommentFn != nil {
		return m.PostCommentFn(prNumber, body)
	}
	return nil
}

func (m *MockGH) ListCollaborators(ctx context.Context, owner, repo string) ([]string, error) {
	if m.ListCollaboratorsFn != nil {
		return m.ListCollaboratorsFn(ctx, owner, repo)
	}
	return nil, nil
}

func (m *MockGH) RegisterWebhook(ctx context.Context, owner, repo, webhookURL, secret string) error {
	if m.RegisterWebhookFn != nil {
		return m.RegisterWebhookFn(ctx, owner, repo, webhookURL, secret)
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
