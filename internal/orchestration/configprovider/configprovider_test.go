package configprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockPrompt struct {
	providerKeyFn func(string) (string, error)
}

func (m *mockPrompt) ProviderKey(provider string) (string, error) {
	return m.providerKeyFn(provider)
}

type mockAuth struct {
	loadFn  func() (map[string]string, error)
	writeFn func(map[string]string) error
	written map[string]string
}

func (m *mockAuth) Load() (map[string]string, error) {
	return m.loadFn()
}

func (m *mockAuth) Write(keys map[string]string) error {
	m.written = keys
	if m.writeFn != nil {
		return m.writeFn(keys)
	}
	return nil
}

type mockK8s struct {
	storedKeys map[string]string
	storeFn    func(context.Context, string, string, map[string]string) error
}

func (m *mockK8s) StoreProviderSecret(ctx context.Context, kubeContext, namespace string, keys map[string]string) error {
	m.storedKeys = keys
	if m.storeFn != nil {
		return m.storeFn(ctx, kubeContext, namespace, keys)
	}
	return nil
}

type runnerOption func(*Runner)

func withPrompt(p PromptClient) runnerOption {
	return func(r *Runner) {
		r.prompt = p
	}
}

func withAuth(a AuthClient) runnerOption {
	return func(r *Runner) {
		r.auth = a
	}
}

func withK8s(k K8sClient) runnerOption {
	return func(r *Runner) {
		r.k8s = k
	}
}

func withMocks(opts ...runnerOption) *Runner {
	r := &Runner{
		prompt: &mockPrompt{
			providerKeyFn: func(string) (string, error) {
				return "", errors.New("unexpected prompt call")
			},
		},
		auth: &mockAuth{
			loadFn: func() (map[string]string, error) {
				return map[string]string{}, nil
			},
		},
		k8s: &mockK8s{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func TestRun_KeyWrittenAndSecretUpdated(t *testing.T) {
	prompt := &mockPrompt{
		providerKeyFn: func(provider string) (string, error) {
			assert.Equal(t, "anthropic", provider)
			return "sk-ant-123", nil
		},
	}
	auth := &mockAuth{
		loadFn: func() (map[string]string, error) {
			return map[string]string{}, nil
		},
	}
	k8s := &mockK8s{}
	r := withMocks(withPrompt(prompt), withAuth(auth), withK8s(k8s))

	err := r.Run(context.Background(), "anthropic", "test-ctx", "test-ns")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"anthropic": "sk-ant-123"}, auth.written)
	assert.Equal(t, map[string]string{"anthropic": "sk-ant-123"}, k8s.storedKeys)
}

func TestRun_ExistingProviderKeysPreserved(t *testing.T) {
	prompt := &mockPrompt{
		providerKeyFn: func(provider string) (string, error) {
			assert.Equal(t, "anthropic", provider)
			return "sk-ant-123", nil
		},
	}
	auth := &mockAuth{
		loadFn: func() (map[string]string, error) {
			return map[string]string{"google": "AIza-existing"}, nil
		},
	}
	k8s := &mockK8s{}
	r := withMocks(withPrompt(prompt), withAuth(auth), withK8s(k8s))

	err := r.Run(context.Background(), "anthropic", "test-ctx", "test-ns")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"anthropic": "sk-ant-123", "google": "AIza-existing"}, auth.written)
	assert.Equal(t, map[string]string{"anthropic": "sk-ant-123", "google": "AIza-existing"}, k8s.storedKeys)
}

func TestRun_BlankKeyReturnsError(t *testing.T) {
	prompt := &mockPrompt{
		providerKeyFn: func(provider string) (string, error) {
			assert.Equal(t, "anthropic", provider)
			return "", errors.New("API key for anthropic cannot be blank")
		},
	}
	auth := &mockAuth{
		loadFn: func() (map[string]string, error) {
			return map[string]string{}, nil
		},
	}
	k8s := &mockK8s{}
	r := withMocks(withPrompt(prompt), withAuth(auth), withK8s(k8s))

	err := r.Run(context.Background(), "anthropic", "test-ctx", "test-ns")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be blank")
	assert.Nil(t, auth.written)
	assert.Nil(t, k8s.storedKeys)
}
