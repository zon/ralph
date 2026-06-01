package argo

import "context"

type MockClient struct {
	ListWorkflowsFunc func(ctx K8sContext) error
	StopWorkflowFunc  func(ctx K8sContext, workflowName string) error
	FollowLogsFunc    func(ctx K8sContext, workflowName string) error
	SubmitYAMLFunc    func(ctx context.Context, workflowYAML string, kubeCtx K8sContext) (string, error)

	ListWorkflowsCalled bool
	StopWorkflowCalled  bool
	FollowLogsCalled    bool
	SubmitYAMLCalled    bool
}

func (m *MockClient) ListWorkflows(ctx K8sContext) error {
	m.ListWorkflowsCalled = true
	if m.ListWorkflowsFunc != nil {
		return m.ListWorkflowsFunc(ctx)
	}
	return nil
}

func (m *MockClient) StopWorkflow(ctx K8sContext, workflowName string) error {
	m.StopWorkflowCalled = true
	if m.StopWorkflowFunc != nil {
		return m.StopWorkflowFunc(ctx, workflowName)
	}
	return nil
}

func (m *MockClient) FollowLogs(ctx K8sContext, workflowName string) error {
	m.FollowLogsCalled = true
	if m.FollowLogsFunc != nil {
		return m.FollowLogsFunc(ctx, workflowName)
	}
	return nil
}

func (m *MockClient) SubmitYAML(ctx context.Context, workflowYAML string, kubeCtx K8sContext) (string, error) {
	m.SubmitYAMLCalled = true
	if m.SubmitYAMLFunc != nil {
		return m.SubmitYAMLFunc(ctx, workflowYAML, kubeCtx)
	}
	return "", nil
}

var _ Client = (*MockClient)(nil)
