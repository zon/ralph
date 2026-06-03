package command

import (
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
)

var errMock = &mockError{"mock error"}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

type mockWorkspaceSetupClient struct {
	setupFunc   func(wksp.WorkspaceFlags) error
	setupCalled bool
}

func (m *mockWorkspaceSetupClient) Setup(flags wksp.WorkspaceFlags) error {
	m.setupCalled = true
	if m.setupFunc != nil {
		return m.setupFunc(flags)
	}
	return nil
}

type mockExecClient struct {
	runFunc    func([]string) error
	runCalled  bool
	runTokens  []string
}

func (m *mockExecClient) Run(tokens []string) error {
	m.runCalled = true
	m.runTokens = tokens
	if m.runFunc != nil {
		return m.runFunc(tokens)
	}
	return nil
}

var mockWksp *mockWorkspaceSetupClient
var mockExec *mockExecClient

type workflowCommandHelper struct{}

type workflowCommandOption func(*WorkflowCommandCmd)

var workflowCommand = &workflowCommandHelper{}

func (h *workflowCommandHelper) withMocks(opts ...workflowCommandOption) *WorkflowCommandCmd {
	mockWksp = &mockWorkspaceSetupClient{}
	mockExec = &mockExecClient{}
	cmd := &WorkflowCommandCmd{
		workspace: mockWksp,
		exec:      mockExec,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (h *workflowCommandHelper) withWorkspace(wc WorkspaceSetupClient) workflowCommandOption {
	return func(cmd *WorkflowCommandCmd) {
		cmd.workspace = wc
		if m, ok := wc.(*mockWorkspaceSetupClient); ok {
			mockWksp = m
		}
	}
}

func (h *workflowCommandHelper) withExec(ec ExecClient) workflowCommandOption {
	return func(cmd *WorkflowCommandCmd) {
		cmd.exec = ec
		if m, ok := ec.(*mockExecClient); ok {
			mockExec = m
		}
	}
}

type workspaceHelper struct{}

var workspace = &workspaceHelper{}

func (h *workspaceHelper) thatFailsSetup() *mockWorkspaceSetupClient {
	return &mockWorkspaceSetupClient{
		setupFunc: func(wksp.WorkspaceFlags) error { return errMock },
	}
}

func (h *workspaceHelper) setupCalled() bool {
	return mockWksp != nil && mockWksp.setupCalled
}

type execHelper struct{}

var exec = &execHelper{}

func (h *execHelper) runCalled() bool {
	return mockExec != nil && mockExec.runCalled
}

type flagsHelper struct{}

var flags = &flagsHelper{}

func (h *flagsHelper) withNoCommand() WorkflowCommandFlags {
	return WorkflowCommandFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
		BotName:     "ralph",
		BotEmail:    "ralph@example.com",
	}
}

func (h *flagsHelper) any() WorkflowCommandFlags {
	return WorkflowCommandFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
		BotName:     "ralph",
		BotEmail:    "ralph@example.com",
		Command:     []string{"echo", "hello"},
	}
}
