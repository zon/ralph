package pulumi

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
)

var errMockFailure = errors.New("mock failure")

type mockConfigLoader struct {
	loadFn func() (*config.RalphConfig, error)
}

func (m *mockConfigLoader) Load() (*config.RalphConfig, error) {
	if m.loadFn != nil {
		return m.loadFn()
	}
	return &config.RalphConfig{
		ConfigPath: "/some/path/.ralph/config.yaml",
		Workflow: config.WorkflowConfig{
			Context:   "my-cluster",
			Namespace: "my-namespace",
		},
	}, nil
}

type mockEnvClient struct {
	getenvFn func(key string) string
	promptFn func(promptMsg string) (string, error)
}

func (m *mockEnvClient) Getenv(key string) string {
	if m.getenvFn != nil {
		return m.getenvFn(key)
	}
	return "pulumi-token-from-env"
}

func (m *mockEnvClient) Prompt(promptMsg string) (string, error) {
	if m.promptFn != nil {
		return m.promptFn(promptMsg)
	}
	return "pulumi-token-from-prompt", nil
}

type mockK8sClient struct {
	getCurrentContextFn    func(ctx context.Context) (K8sContext, error)
	createOrUpdateSecretFn func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
}

func (m *mockK8sClient) GetCurrentContext(ctx context.Context) (K8sContext, error) {
	if m.getCurrentContextFn != nil {
		return m.getCurrentContextFn(ctx)
	}
	return K8sContext{Name: "current-ctx", Namespace: "current-ns"}, nil
}

func (m *mockK8sClient) CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	if m.createOrUpdateSecretFn != nil {
		return m.createOrUpdateSecretFn(ctx, name, namespace, kubeContext, data)
	}
	return nil
}

type mockLogger struct {
	infoFn      func(msg string)
	infofFn     func(format string, args ...interface{})
	successFn   func(msg string)
	successfFn  func(format string, args ...interface{})
	debugfFn    func(format string, args ...interface{})
}

func (m *mockLogger) Info(msg string) {
	if m.infoFn != nil {
		m.infoFn(msg)
	}
}

func (m *mockLogger) Infof(format string, args ...interface{}) {
	if m.infofFn != nil {
		m.infofFn(format, args...)
	}
}

func (m *mockLogger) Success(msg string) {
	if m.successFn != nil {
		m.successFn(msg)
	}
}

func (m *mockLogger) Successf(format string, args ...interface{}) {
	if m.successfFn != nil {
		m.successfFn(format, args...)
	}
}

func (m *mockLogger) Debugf(format string, args ...interface{}) {
	if m.debugfFn != nil {
		m.debugfFn(format, args...)
	}
}

type deps struct {
	configLoader ConfigLoader
	envClient    EnvClient
	k8sClient    K8sClient
	log          Logger
}

type Opt func(*deps)

func withConfigLoader(l ConfigLoader) Opt {
	return func(d *deps) {
		d.configLoader = l
	}
}

func withEnvClient(e EnvClient) Opt {
	return func(d *deps) {
		d.envClient = e
	}
}

func withK8sClient(k K8sClient) Opt {
	return func(d *deps) {
		d.k8sClient = k
	}
}

func withLogger(l Logger) Opt {
	return func(d *deps) {
		d.log = l
	}
}

func newCmd(opts ...Opt) *Cmd {
	d := &deps{
		configLoader: &mockConfigLoader{},
		envClient:    &mockEnvClient{},
		k8sClient:    &mockK8sClient{},
		log:          &mockLogger{},
	}
	for _, opt := range opts {
		opt(d)
	}
	return New(d.configLoader, d.envClient, d.k8sClient, d.log)
}

func TestRun_Success(t *testing.T) {
	var loggedInfo, loggedSuccess, loggedInfof, loggedSuccessf []string
	log := &mockLogger{
		infoFn: func(msg string) {
			loggedInfo = append(loggedInfo, msg)
		},
		successFn: func(msg string) {
			loggedSuccess = append(loggedSuccess, msg)
		},
		infofFn: func(format string, args ...interface{}) {
			loggedInfof = append(loggedInfof, format)
		},
		successfFn: func(format string, args ...interface{}) {
			loggedSuccessf = append(loggedSuccessf, format)
		},
	}

	var createdSecret bool
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			createdSecret = true
			require.Equal(t, "pulumi-credentials", name)
			require.Equal(t, "my-namespace", namespace)
			require.Equal(t, "my-cluster", kubeContext)
			require.Equal(t, "pulumi-token-from-env", data["PULUMI_ACCESS_TOKEN"])
			return nil
		},
	}

	cmd := newCmd(withK8sClient(k8sClient), withLogger(log))
	err := cmd.Run(context.Background(), "", "", "")
	require.NoError(t, err)
	require.True(t, createdSecret)
	require.Contains(t, loggedInfo, "Configuring Pulumi credentials for Ralph remote execution...")
}

func TestRun_ConfigLoadFailure(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(withConfigLoader(cl))
	err := cmd.Run(context.Background(), "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_TokenFromFlag(t *testing.T) {
	envClient := &mockEnvClient{
		getenvFn: func(key string) string {
			t.Fatal("unexpected Getenv call when flag is set")
			return ""
		},
	}
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "flag-token", data["PULUMI_ACCESS_TOKEN"])
			return nil
		},
	}
	cmd := newCmd(withEnvClient(envClient), withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "flag-token", "", "")
	require.NoError(t, err)
}

func TestRun_TokenFromEnv(t *testing.T) {
	envClient := &mockEnvClient{
		getenvFn: func(key string) string {
			require.Equal(t, "PULUMI_ACCESS_TOKEN", key)
			return "env-token"
		},
		promptFn: func(promptMsg string) (string, error) {
			t.Fatal("unexpected Prompt call when env var is set")
			return "", nil
		},
	}
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "env-token", data["PULUMI_ACCESS_TOKEN"])
			return nil
		},
	}
	cmd := newCmd(withEnvClient(envClient), withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRun_TokenFromPrompt(t *testing.T) {
	envClient := &mockEnvClient{
		getenvFn: func(key string) string {
			return ""
		},
		promptFn: func(promptMsg string) (string, error) {
			require.Equal(t, "Pulumi access token: ", promptMsg)
			return "prompt-token", nil
		},
	}
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "prompt-token", data["PULUMI_ACCESS_TOKEN"])
			return nil
		},
	}
	cmd := newCmd(withEnvClient(envClient), withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRun_TokenEmpty(t *testing.T) {
	envClient := &mockEnvClient{
		getenvFn: func(key string) string {
			return ""
		},
		promptFn: func(promptMsg string) (string, error) {
			return "", nil
		},
	}
	cmd := newCmd(withEnvClient(envClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Pulumi access token is required")
}

func TestRun_K8sSecretFailure(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			return errMockFailure
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_KubeContextFailure(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{}, nil
		},
	}
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			return K8sContext{}, errMockFailure
		},
	}
	cmd := newCmd(withConfigLoader(cl), withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_FlagContextPriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			t.Fatal("unexpected GetCurrentContext call")
			return K8sContext{}, nil
		},
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "flag-ctx", kubeContext)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "flag-ctx", "")
	require.NoError(t, err)
}

func TestRun_ConfigContextPriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			t.Fatal("unexpected GetCurrentContext call")
			return K8sContext{}, nil
		},
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "my-cluster", kubeContext)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRun_KubectlContextFallback(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{}, nil
		},
	}
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			return K8sContext{Name: "kubectl-ctx", Namespace: "kubectl-ns"}, nil
		},
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "kubectl-ctx", kubeContext)
			return nil
		},
	}
	cmd := newCmd(withConfigLoader(cl), withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRun_FlagNamespacePriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "flag-ns", namespace)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "flag-ns")
	require.NoError(t, err)
}

func TestRun_ConfigNamespacePriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "my-namespace", namespace)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRun_ConfigPathNamespaceFallback(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{
				ConfigPath: "/some/path/.ralph/config.yaml",
				Workflow:   config.WorkflowConfig{},
			}, nil
		},
	}
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "config", namespace)
			return nil
		},
	}
	cmd := newCmd(withConfigLoader(cl), withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRun_DefaultNamespaceFallback(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{}, nil
		},
	}
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			return K8sContext{Name: "ctx", Namespace: ""}, nil
		},
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "default", namespace)
			return nil
		},
	}
	cmd := newCmd(withConfigLoader(cl), withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRun_KubectlNamespaceFallback(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{}, nil
		},
	}
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			return K8sContext{Name: "ctx", Namespace: "kubectl-ns"}, nil
		},
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "kubectl-ns", namespace)
			return nil
		},
	}
	cmd := newCmd(withConfigLoader(cl), withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), "", "", "")
	require.NoError(t, err)
}
