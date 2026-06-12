package workflow

import (
	"errors"

	"github.com/zon/ralph/internal/project"
)

type MockClient struct {
	SubmitFunc       func(input *project.InputFile, cloneBranch string, debug string, baseBranch string) (string, error)
	FollowLogsFunc   func(workflowName string) error
	PrintLogHintFunc func(workflowName string)

	SubmitCalled       bool
	FollowLogsCalled   bool
	PrintLogHintCalled bool
	LastDebugBranch    string
	LastBaseBranch     string
}

func (m *MockClient) Submit(input *project.InputFile, cloneBranch string, debug string, baseBranch string) (string, error) {
	m.SubmitCalled = true
	m.LastDebugBranch = debug
	m.LastBaseBranch = baseBranch
	if m.SubmitFunc != nil {
		return m.SubmitFunc(input, cloneBranch, debug, baseBranch)
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
		SubmitFunc: func(input *project.InputFile, cloneBranch string, debug string, baseBranch string) (string, error) {
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
