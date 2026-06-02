package github

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

type mockGitHubClient struct {
	getRepoFn              func(ctx context.Context) (string, string, error)
	generateAppJWTFn       func(appID string, privateKeyPEM []byte) (string, error)
	getInstallationIDFn    func(ctx context.Context, jwtToken, owner, repo string) (int64, error)
	getInstallationTokenFn func(ctx context.Context, jwtToken string, installationID int64) (string, error)
}

func (m *mockGitHubClient) GetRepo(ctx context.Context) (string, string, error) {
	if m.getRepoFn != nil {
		return m.getRepoFn(ctx)
	}
	return "test-owner", "test-repo", nil
}

func (m *mockGitHubClient) GenerateAppJWT(appID string, privateKeyPEM []byte) (string, error) {
	if m.generateAppJWTFn != nil {
		return m.generateAppJWTFn(appID, privateKeyPEM)
	}
	return "test-jwt", nil
}

func (m *mockGitHubClient) GetInstallationID(ctx context.Context, jwtToken, owner, repo string) (int64, error) {
	if m.getInstallationIDFn != nil {
		return m.getInstallationIDFn(ctx, jwtToken, owner, repo)
	}
	return 12345, nil
}

func (m *mockGitHubClient) GetInstallationToken(ctx context.Context, jwtToken string, installationID int64) (string, error) {
	if m.getInstallationTokenFn != nil {
		return m.getInstallationTokenFn(ctx, jwtToken, installationID)
	}
	return "test-installation-token", nil
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
	infoFn    func(msg string)
	infofFn   func(format string, args ...interface{})
	successFn func(msg string)
	successfFn func(format string, args ...interface{})
	debugfFn  func(format string, args ...interface{})
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
	ghClient     GitHubClient
	k8sClient    K8sClient
	log          Logger
}

type Opt func(*deps)

func withConfigLoader(l ConfigLoader) Opt {
	return func(d *deps) {
		d.configLoader = l
	}
}

func withGitHubClient(g GitHubClient) Opt {
	return func(d *deps) {
		d.ghClient = g
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
		ghClient:     &mockGitHubClient{},
		k8sClient:    &mockK8sClient{},
		log:          &mockLogger{},
	}
	for _, opt := range opts {
		opt(d)
	}
	return New(d.configLoader, d.ghClient, d.k8sClient, d.log)
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
			require.Equal(t, "github-credentials", name)
			require.Equal(t, "my-namespace", namespace)
			require.Equal(t, "my-cluster", kubeContext)
			require.Equal(t, "2966665", data["app-id"])
			require.Equal(t, "test-private-key", data["private-key"])
			return nil
		},
	}

	cmd := newCmd(withK8sClient(k8sClient), withLogger(log))
	err := cmd.Run(context.Background(), []byte("test-private-key"), "", "")
	require.NoError(t, err)
	require.True(t, createdSecret)
	require.Contains(t, loggedInfo, "Configuring GitHub App credentials for Ralph remote execution...")
	require.Contains(t, loggedSuccess, "Credentials validated successfully")
	require.Contains(t, loggedInfo, "Note: GitHub App credentials are not tied to any user account.")
}

func TestRun_ConfigLoadFailure(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(withConfigLoader(cl))
	err := cmd.Run(context.Background(), []byte("key"), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_GetRepoFailure(t *testing.T) {
	gh := &mockGitHubClient{
		getRepoFn: func(ctx context.Context) (string, string, error) {
			return "", "", errMockFailure
		},
	}
	cmd := newCmd(withGitHubClient(gh))
	err := cmd.Run(context.Background(), []byte("key"), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_EmptyPrivateKey(t *testing.T) {
	cmd := newCmd()
	err := cmd.Run(context.Background(), []byte{}, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "private key file is empty")
}

func TestRun_ValidateCredentialsJWTFailure(t *testing.T) {
	gh := &mockGitHubClient{
		generateAppJWTFn: func(appID string, privateKeyPEM []byte) (string, error) {
			return "", errMockFailure
		},
	}
	cmd := newCmd(withGitHubClient(gh))
	err := cmd.Run(context.Background(), []byte("key"), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_ValidateCredentialsInstallationIDFailure(t *testing.T) {
	gh := &mockGitHubClient{
		getInstallationIDFn: func(ctx context.Context, jwtToken, owner, repo string) (int64, error) {
			return 0, errMockFailure
		},
	}
	cmd := newCmd(withGitHubClient(gh))
	err := cmd.Run(context.Background(), []byte("key"), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_ValidateCredentialsInstallationTokenFailure(t *testing.T) {
	gh := &mockGitHubClient{
		getInstallationTokenFn: func(ctx context.Context, jwtToken string, installationID int64) (string, error) {
			return "", errMockFailure
		},
	}
	cmd := newCmd(withGitHubClient(gh))
	err := cmd.Run(context.Background(), []byte("key"), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRun_K8sSecretFailure(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			return errMockFailure
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.Run(context.Background(), []byte("key"), "", "")
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
	err := cmd.Run(context.Background(), []byte("key"), "", "")
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
	err := cmd.Run(context.Background(), []byte("key"), "flag-ctx", "")
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
	err := cmd.Run(context.Background(), []byte("key"), "", "")
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
	err := cmd.Run(context.Background(), []byte("key"), "", "")
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
	err := cmd.Run(context.Background(), []byte("key"), "", "flag-ns")
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
	err := cmd.Run(context.Background(), []byte("key"), "", "")
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
	err := cmd.Run(context.Background(), []byte("key"), "", "")
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
	err := cmd.Run(context.Background(), []byte("key"), "", "")
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
	err := cmd.Run(context.Background(), []byte("key"), "", "")
	require.NoError(t, err)
}

func TestRun_PrivateKeyValidation(t *testing.T) {
	t.Run("empty bytes", func(t *testing.T) {
		cmd := newCmd()
		err := cmd.Run(context.Background(), []byte{}, "", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty")
	})

	t.Run("nil bytes", func(t *testing.T) {
		cmd := newCmd()
		err := cmd.Run(context.Background(), nil, "", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty")
	})
}
