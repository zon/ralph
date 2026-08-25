package argo

import "context"

type MockClient struct {
	ListWorkflowsFunc     func(ctx K8sContext) error
	ListWorkflowNamesFunc func(ctx K8sContext) ([]string, error)
	StopWorkflowFunc      func(ctx K8sContext, workflowName string) error
	LogsFunc              func(ctx K8sContext, workflowName string, follow bool) error
	SubmitYAMLFunc        func(ctx context.Context, workflowYAML string, kubeCtx K8sContext) (string, error)

	SubmitYAMLCalled bool
}

func (m *MockClient) ListWorkflows(ctx K8sContext) error {
	if m.ListWorkflowsFunc != nil {
		return m.ListWorkflowsFunc(ctx)
	}
	return nil
}

func (m *MockClient) StopWorkflow(ctx K8sContext, workflowName string) error {
	if m.StopWorkflowFunc != nil {
		return m.StopWorkflowFunc(ctx, workflowName)
	}
	return nil
}

func (m *MockClient) ListWorkflowNames(ctx K8sContext) ([]string, error) {
	if m.ListWorkflowNamesFunc != nil {
		return m.ListWorkflowNamesFunc(ctx)
	}
	return nil, nil
}

func (m *MockClient) Logs(ctx K8sContext, workflowName string, follow bool) error {
	if m.LogsFunc != nil {
		return m.LogsFunc(ctx, workflowName, follow)
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
