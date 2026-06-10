package k8s

import "context"

type MockClient struct {
	GetCurrentContextFunc       func(ctx context.Context) (Context, error)
	CreateOrUpdateConfigMapFunc func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
	CreateOrUpdateSecretFunc    func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
	SecretExistsFunc            func(ctx context.Context, name, namespace, kubeContext string) (bool, error)
	GetConfigMapDataFunc        func(ctx context.Context, name, namespace, kubeContext string) (string, error)
}

func (m *MockClient) GetCurrentContext(ctx context.Context) (Context, error) {
	if m.GetCurrentContextFunc != nil {
		return m.GetCurrentContextFunc(ctx)
	}
	return Context{}, nil
}

func (m *MockClient) CreateOrUpdateConfigMap(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	if m.CreateOrUpdateConfigMapFunc != nil {
		return m.CreateOrUpdateConfigMapFunc(ctx, name, namespace, kubeContext, data)
	}
	return nil
}

func (m *MockClient) CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	if m.CreateOrUpdateSecretFunc != nil {
		return m.CreateOrUpdateSecretFunc(ctx, name, namespace, kubeContext, data)
	}
	return nil
}

func (m *MockClient) SecretExists(ctx context.Context, name, namespace, kubeContext string) (bool, error) {
	if m.SecretExistsFunc != nil {
		return m.SecretExistsFunc(ctx, name, namespace, kubeContext)
	}
	return false, nil
}

func (m *MockClient) GetConfigMapData(ctx context.Context, name, namespace, kubeContext string) (string, error) {
	if m.GetConfigMapDataFunc != nil {
		return m.GetConfigMapDataFunc(ctx, name, namespace, kubeContext)
	}
	return "", nil
}

var _ Client = (*MockClient)(nil)
