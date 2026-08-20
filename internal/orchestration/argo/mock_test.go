package argo

import "errors"

var errMock = errors.New("mock error")

type mockArgoClient struct {
	listFunc                func(K8sContext) error
	listCalled              bool
	listCtx                 K8sContext
	stopFunc                func(K8sContext, string) error
	stopCalled              bool
	stopCtx                 K8sContext
	listWorkflowNamesFunc   func(K8sContext) ([]string, error)
	listWorkflowNamesCalled bool
	listWorkflowNamesCtx    K8sContext
	logsFunc                func(K8sContext, string, bool) error
	logsCalled              bool
	logsCtx                 K8sContext
	logsWorkflowName        string
	logsFollow              bool
}

func (m *mockArgoClient) List(ctx K8sContext) error {
	m.listCalled = true
	m.listCtx = ctx
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return nil
}

func (m *mockArgoClient) Stop(ctx K8sContext, workflowName string) error {
	m.stopCalled = true
	m.stopCtx = ctx
	if m.stopFunc != nil {
		return m.stopFunc(ctx, workflowName)
	}
	return nil
}

func (m *mockArgoClient) ListWorkflowNames(ctx K8sContext) ([]string, error) {
	m.listWorkflowNamesCalled = true
	m.listWorkflowNamesCtx = ctx
	if m.listWorkflowNamesFunc != nil {
		return m.listWorkflowNamesFunc(ctx)
	}
	return nil, nil
}

func (m *mockArgoClient) Logs(ctx K8sContext, workflowName string, follow bool) error {
	m.logsCalled = true
	m.logsCtx = ctx
	m.logsWorkflowName = workflowName
	m.logsFollow = follow
	if m.logsFunc != nil {
		return m.logsFunc(ctx, workflowName, follow)
	}
	return nil
}

type mockContextClient struct {
	resolveFunc func(string, string) (K8sContext, error)
}

func (m *mockContextClient) Resolve(flagContext, flagNamespace string) (K8sContext, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(flagContext, flagNamespace)
	}
	return ctx.any(), nil
}

var mockArgo *mockArgoClient
var mockCtx *mockContextClient

type argoHelper struct{}
type argoOption func(*ArgoCmd)

var argo = &argoHelper{}

func (h *argoHelper) withMocks(opts ...argoOption) *ArgoCmd {
	mockArgo = &mockArgoClient{}
	mockCtx = &mockContextClient{}
	cmd := &ArgoCmd{Argo: mockArgo, Ctx: mockCtx}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (h *argoHelper) withContext(cc ContextClient) argoOption {
	return func(cmd *ArgoCmd) {
		cmd.Ctx = cc
		if m, ok := cc.(*mockContextClient); ok {
			mockCtx = m
		}
	}
}

func (h *argoHelper) withWorkflowNames(names ...string) argoOption {
	return func(cmd *ArgoCmd) {
		mockArgo.listWorkflowNamesFunc = func(K8sContext) ([]string, error) {
			return names, nil
		}
	}
}

func (h *argoHelper) withNoWorkflowNames() argoOption {
	return func(cmd *ArgoCmd) {
		mockArgo.listWorkflowNamesFunc = func(K8sContext) ([]string, error) {
			return nil, nil
		}
	}
}

func (h *argoHelper) withListNamesFailure() argoOption {
	return func(cmd *ArgoCmd) {
		mockArgo.listWorkflowNamesFunc = func(K8sContext) ([]string, error) {
			return nil, errMock
		}
	}
}

func (h *argoHelper) withLogsFailure() argoOption {
	return func(cmd *ArgoCmd) {
		mockArgo.logsFunc = func(K8sContext, string, bool) error {
			return errMock
		}
	}
}

type ctxHelper struct{}

var ctx = &ctxHelper{}

func (h *ctxHelper) any() K8sContext {
	return K8sContext{Name: "test-cluster", Namespace: "resolved-ns"}
}

func (h *ctxHelper) thatFails() *mockContextClient {
	return &mockContextClient{
		resolveFunc: func(string, string) (K8sContext, error) {
			return K8sContext{}, errMock
		},
	}
}

type argoClientHelper struct{}

var argoClient = &argoClientHelper{}

func (h *argoClientHelper) listCalled() bool {
	return mockArgo != nil && mockArgo.listCalled
}

func (h *argoClientHelper) listContext() K8sContext {
	if mockArgo == nil {
		return K8sContext{}
	}
	return mockArgo.listCtx
}

func (h *argoClientHelper) stopCalled() bool {
	return mockArgo != nil && mockArgo.stopCalled
}

func (h *argoClientHelper) stopContext() K8sContext {
	if mockArgo == nil {
		return K8sContext{}
	}
	return mockArgo.stopCtx
}

func (h *argoClientHelper) listWorkflowNamesCalled() bool {
	return mockArgo != nil && mockArgo.listWorkflowNamesCalled
}

func (h *argoClientHelper) listWorkflowNamesContext() K8sContext {
	if mockArgo == nil {
		return K8sContext{}
	}
	return mockArgo.listWorkflowNamesCtx
}

func (h *argoClientHelper) logsCalled() bool {
	return mockArgo != nil && mockArgo.logsCalled
}

func (h *argoClientHelper) logsContext() K8sContext {
	if mockArgo == nil {
		return K8sContext{}
	}
	return mockArgo.logsCtx
}

func (h *argoClientHelper) loggedWorkflow() string {
	if mockArgo == nil {
		return ""
	}
	return mockArgo.logsWorkflowName
}

func (h *argoClientHelper) loggedFollow() bool {
	return mockArgo != nil && mockArgo.logsFollow
}

type flagsHelper struct{}

var flags = &flagsHelper{}

func (h *flagsHelper) anyList() ListFlags {
	return ListFlags{Context: "test-ctx", Namespace: "test-ns"}
}

func (h *flagsHelper) anyStop() StopFlags {
	return StopFlags{Context: "test-ctx", Namespace: "test-ns", WorkflowName: "test-workflow"}
}

func (h *flagsHelper) anyLogs() LogsFlags {
	return LogsFlags{Context: "test-ctx", Namespace: "test-ns", WorkflowName: "test-workflow"}
}

func (h *flagsHelper) logsWithoutName() LogsFlags {
	return LogsFlags{Context: "test-ctx", Namespace: "test-ns"}
}

func (h *flagsHelper) followingLogs() LogsFlags {
	return LogsFlags{Context: "test-ctx", Namespace: "test-ns", WorkflowName: "test-workflow", Follow: true}
}
