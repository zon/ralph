package argo

import "errors"

var errMock = errors.New("mock error")

type mockArgoClient struct {
	listFunc    func(K8sContext) error
	listCalled  bool
	stopFunc    func(K8sContext, string) error
	stopCalled  bool
}

func (m *mockArgoClient) List(ctx K8sContext) error {
	m.listCalled = true
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return nil
}

func (m *mockArgoClient) Stop(ctx K8sContext, workflowName string) error {
	m.stopCalled = true
	if m.stopFunc != nil {
		return m.stopFunc(ctx, workflowName)
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
	return K8sContext{}, nil
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

type ctxHelper struct{}

var ctx = &ctxHelper{}

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

func (h *argoClientHelper) stopCalled() bool {
	return mockArgo != nil && mockArgo.stopCalled
}

type flagsHelper struct{}

var flags = &flagsHelper{}

func (h *flagsHelper) anyList() ListFlags {
	return ListFlags{Context: "test-ctx", Namespace: "test-ns"}
}

func (h *flagsHelper) anyStop() StopFlags {
	return StopFlags{Context: "test-ctx", Namespace: "test-ns", WorkflowName: "test-workflow"}
}
