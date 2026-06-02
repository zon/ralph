package list

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
)

var errMockFailure = errors.New("mock failure")

type mockConfigLoader struct {
	loadFn func() (*config.RalphConfig, error)
}

func (m *mockConfigLoader) Load() (*config.RalphConfig, error) {
	if m.loadFn != nil {
		return m.loadFn()
	}
	return &config.RalphConfig{}, nil
}

type mockK8sClient struct {
	getCurrentContextFn func(ctx context.Context) (k8s.Context, error)
}

func (m *mockK8sClient) GetCurrentContext(ctx context.Context) (k8s.Context, error) {
	if m.getCurrentContextFn != nil {
		return m.getCurrentContextFn(ctx)
	}
	return k8s.Context{Name: "kubectl-ctx", Namespace: "kubectl-ns"}, nil
}

type mockArgoClient struct {
	listWorkflowsFn func(ctx KubeContext) error
}

func (m *mockArgoClient) ListWorkflows(ctx KubeContext) error {
	if m.listWorkflowsFn != nil {
		return m.listWorkflowsFn(ctx)
	}
	return nil
}

type deps struct {
	configLoader ConfigLoader
	k8sClient    K8sClient
	argoClient   ArgoClient
}

type Opt func(*deps)

func withConfigLoader(c ConfigLoader) Opt {
	return func(d *deps) {
		d.configLoader = c
	}
}

func withK8sClient(c K8sClient) Opt {
	return func(d *deps) {
		d.k8sClient = c
	}
}

func withArgoClient(c ArgoClient) Opt {
	return func(d *deps) {
		d.argoClient = c
	}
}

func newList(opts ...Opt) *List {
	d := &deps{
		configLoader: &mockConfigLoader{},
		k8sClient:    &mockK8sClient{},
		argoClient:   &mockArgoClient{},
	}
	for _, opt := range opts {
		opt(d)
	}
	return New(d.configLoader, d.k8sClient, d.argoClient)
}

func TestRun_Success(t *testing.T) {
	listCalled := false
	l := newList(
		withK8sClient(&mockK8sClient{
			getCurrentContextFn: func(ctx context.Context) (k8s.Context, error) {
				return k8s.Context{Name: "kubectl-ctx"}, nil
			},
		}),
		withArgoClient(&mockArgoClient{
			listWorkflowsFn: func(ctx KubeContext) error {
				listCalled = true
				require.Equal(t, "kubectl-ctx", ctx.Name)
				require.Equal(t, "default", ctx.Namespace)
				return nil
			},
		}),
	)
	err := l.Run(context.Background(), "")
	require.NoError(t, err)
	require.True(t, listCalled)
}

func TestRun_ConfigLoadFails(t *testing.T) {
	l := newList(
		withConfigLoader(&mockConfigLoader{
			loadFn: func() (*config.RalphConfig, error) {
				return nil, errMockFailure
			},
		}),
	)
	err := l.Run(context.Background(), "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load config")
}

func TestRun_ResolveKubeContextFails(t *testing.T) {
	l := newList(
		withK8sClient(&mockK8sClient{
			getCurrentContextFn: func(ctx context.Context) (k8s.Context, error) {
				return k8s.Context{}, errMockFailure
			},
		}),
	)
	err := l.Run(context.Background(), "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get current Kubernetes context")
}

func TestRun_ListWorkflowsFails(t *testing.T) {
	l := newList(
		withArgoClient(&mockArgoClient{
			listWorkflowsFn: func(ctx KubeContext) error {
				return errMockFailure
			},
		}),
	)
	err := l.Run(context.Background(), "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestResolveKubeContext_UsesFlagContext(t *testing.T) {
	l := newList()
	kc, err := l.resolveKubeContext(context.Background(), &config.RalphConfig{}, "my-ctx")
	require.NoError(t, err)
	require.Equal(t, "my-ctx", kc.Name)
	require.Equal(t, "default", kc.Namespace)
}

func TestResolveKubeContext_UsesConfigContext(t *testing.T) {
	l := newList()
	kc, err := l.resolveKubeContext(context.Background(), &config.RalphConfig{
		Workflow: config.WorkflowConfig{
			Context: "config-ctx",
		},
	}, "")
	require.NoError(t, err)
	require.Equal(t, "config-ctx", kc.Name)
	require.Equal(t, "default", kc.Namespace)
}

func TestResolveKubeContext_UsesKubectlCurrentContext(t *testing.T) {
	l := newList(
		withK8sClient(&mockK8sClient{
			getCurrentContextFn: func(ctx context.Context) (k8s.Context, error) {
				return k8s.Context{Name: "kubectl-ctx"}, nil
			},
		}),
	)
	kc, err := l.resolveKubeContext(context.Background(), &config.RalphConfig{}, "")
	require.NoError(t, err)
	require.Equal(t, "kubectl-ctx", kc.Name)
	require.Equal(t, "default", kc.Namespace)
}

func TestResolveKubeContext_NamespaceFromConfig(t *testing.T) {
	l := newList()
	kc, err := l.resolveKubeContext(context.Background(), &config.RalphConfig{
		Workflow: config.WorkflowConfig{
			Namespace: "config-ns",
		},
	}, "")
	require.NoError(t, err)
	require.Equal(t, "config-ns", kc.Namespace)
}

func TestResolveKubeContext_NamespaceFromConfigPath(t *testing.T) {
	l := newList()
	kc, err := l.resolveKubeContext(context.Background(), &config.RalphConfig{
		ConfigPath: "/some/path/.ralph/config.yaml",
	}, "")
	require.NoError(t, err)
	require.Equal(t, "config", kc.Namespace)
}

func TestResolveKubeContext_NamespaceFromKubectl(t *testing.T) {
	l := newList(
		withK8sClient(&mockK8sClient{
			getCurrentContextFn: func(ctx context.Context) (k8s.Context, error) {
				return k8s.Context{Name: "my-ctx", Namespace: "kubectl-ns"}, nil
			},
		}),
	)
	kc, err := l.resolveKubeContext(context.Background(), &config.RalphConfig{}, "")
	require.NoError(t, err)
	require.Equal(t, "kubectl-ns", kc.Namespace)
}

func TestResolveKubeContext_DefaultNamespace(t *testing.T) {
	l := newList(
		withK8sClient(&mockK8sClient{
			getCurrentContextFn: func(ctx context.Context) (k8s.Context, error) {
				return k8s.Context{Name: "my-ctx"}, nil
			},
		}),
	)
	kc, err := l.resolveKubeContext(context.Background(), &config.RalphConfig{}, "")
	require.NoError(t, err)
	require.Equal(t, "default", kc.Namespace)
}

func TestResolveKubeContext_KubectlError(t *testing.T) {
	l := newList(
		withK8sClient(&mockK8sClient{
			getCurrentContextFn: func(ctx context.Context) (k8s.Context, error) {
				return k8s.Context{}, errMockFailure
			},
		}),
	)
	_, err := l.resolveKubeContext(context.Background(), &config.RalphConfig{}, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get current Kubernetes context")
}

func TestRun_UsesFlagContext(t *testing.T) {
	listCalled := false
	l := newList(
		withArgoClient(&mockArgoClient{
			listWorkflowsFn: func(ctx KubeContext) error {
				listCalled = true
				require.Equal(t, "flag-ctx", ctx.Name)
				return nil
			},
		}),
	)
	err := l.Run(context.Background(), "flag-ctx")
	require.NoError(t, err)
	require.True(t, listCalled)
}

func TestRun_UsesConfigNamespace(t *testing.T) {
	listCalled := false
	l := newList(
		withConfigLoader(&mockConfigLoader{
			loadFn: func() (*config.RalphConfig, error) {
				return &config.RalphConfig{
					Workflow: config.WorkflowConfig{
						Namespace: "cfg-ns",
					},
				}, nil
			},
		}),
		withK8sClient(&mockK8sClient{
			getCurrentContextFn: func(ctx context.Context) (k8s.Context, error) {
				return k8s.Context{Name: "some-ctx"}, nil
			},
		}),
		withArgoClient(&mockArgoClient{
			listWorkflowsFn: func(ctx KubeContext) error {
				listCalled = true
				require.Equal(t, "cfg-ns", ctx.Namespace)
				return nil
			},
		}),
	)
	err := l.Run(context.Background(), "")
	require.NoError(t, err)
	require.True(t, listCalled)
}
