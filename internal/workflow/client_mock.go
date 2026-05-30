package workflow

import (
	"errors"

	"github.com/zon/ralph/internal/project"
)

type MockClient struct {
	SubmitFunc       func(proj *project.Project, cloneBranch string) (string, error)
	FollowLogsFunc   func(workflowName string) error
	PrintLogHintFunc func(workflowName string)

	SubmitCalled       bool
	FollowLogsCalled   bool
	PrintLogHintCalled bool
}

func (m *MockClient) Submit(proj *project.Project, cloneBranch string) (string, error) {
	m.SubmitCalled = true
	if m.SubmitFunc != nil {
		return m.SubmitFunc(proj, cloneBranch)
	}
	return "test-workflow", nil
}

func (m *MockClient) FollowLogs(workflowName string) error {
	m.FollowLogsCalled = true
	if m.FollowLogsFunc != nil {
		return m.FollowLogsFunc(workflowName)
	}
	return nil
}

func (m *MockClient) PrintLogHint(workflowName string) {
	m.PrintLogHintCalled = true
	if m.PrintLogHintFunc != nil {
		m.PrintLogHintFunc(workflowName)
	}
}

func ThatFailsOnSubmit() *MockClient {
	return &MockClient{
		SubmitFunc: func(proj *project.Project, cloneBranch string) (string, error) {
			return "", errors.New("submit failed")
		},
	}
}

func ThatFailsOnFollow() *MockClient {
	return &MockClient{
		FollowLogsFunc: func(workflowName string) error {
			return errors.New("follow failed")
		},
	}
}
