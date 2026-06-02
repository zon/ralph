package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/webhookconfig"
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
	getRepoFn         func(ctx context.Context) (string, string, error)
	registerWebhookFn func(ctx context.Context, owner, repo, webhookURL, secret string) error
}

func (m *mockGitHubClient) GetRepo(ctx context.Context) (string, string, error) {
	if m.getRepoFn != nil {
		return m.getRepoFn(ctx)
	}
	return "test-owner", "test-repo", nil
}

func (m *mockGitHubClient) RegisterWebhook(ctx context.Context, owner, repo, webhookURL, secret string) error {
	if m.registerWebhookFn != nil {
		return m.registerWebhookFn(ctx, owner, repo, webhookURL, secret)
	}
	return nil
}

type mockK8sClient struct {
	getCurrentContextFn        func(ctx context.Context) (K8sContext, error)
	createOrUpdateConfigMapFn  func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
	createOrUpdateSecretFn     func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error
}

func (m *mockK8sClient) GetCurrentContext(ctx context.Context) (K8sContext, error) {
	if m.getCurrentContextFn != nil {
		return m.getCurrentContextFn(ctx)
	}
	return K8sContext{Name: "current-ctx", Namespace: "current-ns"}, nil
}

func (m *mockK8sClient) CreateOrUpdateConfigMap(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	if m.createOrUpdateConfigMapFn != nil {
		return m.createOrUpdateConfigMapFn(ctx, name, namespace, kubeContext, data)
	}
	return nil
}

func (m *mockK8sClient) CreateOrUpdateSecret(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
	if m.createOrUpdateSecretFn != nil {
		return m.createOrUpdateSecretFn(ctx, name, namespace, kubeContext, data)
	}
	return nil
}

type mockConfigMapReader struct {
	readWebhookConfigFn func(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error)
}

func (m *mockConfigMapReader) ReadWebhookConfig(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error) {
	if m.readWebhookConfigFn != nil {
		return m.readWebhookConfigFn(ctx, namespace, kubeContext)
	}
	return &webhookconfig.AppConfig{
		Port: 8080,
		Repos: []webhookconfig.RepoConfig{
			{Owner: "existing-owner", Name: "existing-repo", Namespace: "existing-ns"},
		},
	}, nil
}

type mockAppConfigLoader struct {
	loadAppConfigFn func(path string) (*webhookconfig.AppConfig, error)
}

func (m *mockAppConfigLoader) LoadAppConfig(path string) (*webhookconfig.AppConfig, error) {
	if m.loadAppConfigFn != nil {
		return m.loadAppConfigFn(path)
	}
	return &webhookconfig.AppConfig{Port: 9090}, nil
}

type mockAppConfigBuilder struct {
	buildFn func(ctx context.Context, base, updates *webhookconfig.AppConfig, repoOwner, repoName, repoNamespace string) webhookconfig.AppConfig
}

func (m *mockAppConfigBuilder) Build(ctx context.Context, base, updates *webhookconfig.AppConfig, repoOwner, repoName, repoNamespace string) webhookconfig.AppConfig {
	if m.buildFn != nil {
		return m.buildFn(ctx, base, updates, repoOwner, repoName, repoNamespace)
	}
	return webhookconfig.AppConfig{
		Port:      8080,
		RalphUser: "ralph-zon[bot]",
	}
}

type mockSecretBuilder struct {
	buildSecretsFn func(appCfg *webhookconfig.AppConfig) (*webhookconfig.Secrets, error)
}

func (m *mockSecretBuilder) BuildSecrets(appCfg *webhookconfig.AppConfig) (*webhookconfig.Secrets, error) {
	if m.buildSecretsFn != nil {
		return m.buildSecretsFn(appCfg)
	}
	return &webhookconfig.Secrets{
		Repos: []webhookconfig.RepoSecret{
			{Owner: "test-owner", Name: "test-repo", WebhookSecret: "test-secret"},
		},
	}, nil
}

type mockLogger struct {
	infoFn     func(msg string)
	infofFn    func(format string, args ...interface{})
	warnfFn    func(format string, args ...interface{})
	successFn  func(msg string)
	successfFn func(format string, args ...interface{})
	debugfFn   func(format string, args ...interface{})
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

func (m *mockLogger) Warnf(format string, args ...interface{}) {
	if m.warnfFn != nil {
		m.warnfFn(format, args...)
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
	configLoader     ConfigLoader
	ghClient         GitHubClient
	k8sClient        K8sClient
	configMapReader  ConfigMapReader
	appConfigLoader  AppConfigLoader
	appConfigBuilder AppConfigBuilder
	secretBuilder    SecretBuilder
	log              Logger
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

func withConfigMapReader(r ConfigMapReader) Opt {
	return func(d *deps) {
		d.configMapReader = r
	}
}

func withAppConfigLoader(l AppConfigLoader) Opt {
	return func(d *deps) {
		d.appConfigLoader = l
	}
}

func withAppConfigBuilder(b AppConfigBuilder) Opt {
	return func(d *deps) {
		d.appConfigBuilder = b
	}
}

func withSecretBuilder(s SecretBuilder) Opt {
	return func(d *deps) {
		d.secretBuilder = s
	}
}

func withLogger(l Logger) Opt {
	return func(d *deps) {
		d.log = l
	}
}

func newCmd(opts ...Opt) *Cmd {
	d := &deps{
		configLoader:     &mockConfigLoader{},
		ghClient:         &mockGitHubClient{},
		k8sClient:        &mockK8sClient{},
		configMapReader:  &mockConfigMapReader{},
		appConfigLoader:  &mockAppConfigLoader{},
		appConfigBuilder: &mockAppConfigBuilder{},
		secretBuilder:    &mockSecretBuilder{},
		log:              &mockLogger{},
	}
	for _, opt := range opts {
		opt(d)
	}
	return New(d.configLoader, d.ghClient, d.k8sClient, d.configMapReader, d.appConfigLoader, d.appConfigBuilder, d.secretBuilder, d.log)
}

// RunConfig tests

func TestRunConfig_Success(t *testing.T) {
	var loggedInfo, loggedInfof []string
	log := &mockLogger{
		infoFn: func(msg string) {
			loggedInfo = append(loggedInfo, msg)
		},
		infofFn: func(format string, args ...interface{}) {
			loggedInfof = append(loggedInfof, format)
		},
	}

	var createdConfigMap bool
	k8sClient := &mockK8sClient{
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			createdConfigMap = true
			require.Equal(t, "webhook-config", name)
			require.Equal(t, "my-namespace", namespace)
			require.Equal(t, "my-cluster", kubeContext)
			require.Contains(t, data["config.yaml"], "port: 8080")
			return nil
		},
	}

	cmd := newCmd(withK8sClient(k8sClient), withLogger(log))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.NoError(t, err)
	require.True(t, createdConfigMap)
	require.Contains(t, loggedInfo, "Provisioning webhook-config configmap...")
}

func TestRunConfig_ConfigLoadFailure(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(withConfigLoader(cl))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRunConfig_ConfigMapReadFailure_Warns(t *testing.T) {
	var warned bool
	log := &mockLogger{
		warnfFn: func(format string, args ...interface{}) {
			warned = true
		},
	}
	reader := &mockConfigMapReader{
		readWebhookConfigFn: func(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(withConfigMapReader(reader), withLogger(log))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.NoError(t, err)
	require.True(t, warned)
}

func TestRunConfig_AppConfigLoadFailure_Warns(t *testing.T) {
	var warned bool
	log := &mockLogger{
		warnfFn: func(format string, args ...interface{}) {
			warned = true
		},
	}
	loader := &mockAppConfigLoader{
		loadAppConfigFn: func(path string) (*webhookconfig.AppConfig, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(withAppConfigLoader(loader), withLogger(log))
	err := cmd.RunConfig(context.Background(), "/some/path", "", "")
	require.NoError(t, err)
	require.True(t, warned)
}

func TestRunConfig_RepoDetectionFailure_Warns(t *testing.T) {
	var warned bool
	log := &mockLogger{
		warnfFn: func(format string, args ...interface{}) {
			warned = true
		},
	}
	gh := &mockGitHubClient{
		getRepoFn: func(ctx context.Context) (string, string, error) {
			return "", "", errMockFailure
		},
	}
	cmd := newCmd(withGitHubClient(gh), withLogger(log))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.NoError(t, err)
	require.True(t, warned)
}

func TestRunConfig_ConfigMapWriteFailure(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			return errMockFailure
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRunConfig_KubeContextFailure(t *testing.T) {
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
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRunConfig_FlagContextPriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			t.Fatal("unexpected GetCurrentContext call")
			return K8sContext{}, nil
		},
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "flag-ctx", kubeContext)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.RunConfig(context.Background(), "", "flag-ctx", "")
	require.NoError(t, err)
}

func TestRunConfig_ConfigContextPriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			t.Fatal("unexpected GetCurrentContext call")
			return K8sContext{}, nil
		},
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "my-cluster", kubeContext)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRunConfig_KubectlContextFallback(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{}, nil
		},
	}
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			return K8sContext{Name: "kubectl-ctx", Namespace: "kubectl-ns"}, nil
		},
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "kubectl-ctx", kubeContext)
			return nil
		},
	}
	cmd := newCmd(withConfigLoader(cl), withK8sClient(k8sClient))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRunConfig_FlagNamespacePriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "flag-ns", namespace)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.RunConfig(context.Background(), "", "", "flag-ns")
	require.NoError(t, err)
}

func TestRunConfig_ConfigNamespacePriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "my-namespace", namespace)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRunConfig_ConfigPathNamespaceFallback(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{
				ConfigPath: "/some/path/.ralph/config.yaml",
				Workflow:   config.WorkflowConfig{},
			}, nil
		},
	}
	k8sClient := &mockK8sClient{
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "config", namespace)
			return nil
		},
	}
	cmd := newCmd(withConfigLoader(cl), withK8sClient(k8sClient))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRunConfig_DefaultNamespaceFallback(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{}, nil
		},
	}
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			return K8sContext{Name: "ctx", Namespace: ""}, nil
		},
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "default", namespace)
			return nil
		},
	}
	cmd := newCmd(withConfigLoader(cl), withK8sClient(k8sClient))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.NoError(t, err)
}

func TestRunConfig_KubectlNamespaceFallback(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{}, nil
		},
	}
	k8sClient := &mockK8sClient{
		getCurrentContextFn: func(ctx context.Context) (K8sContext, error) {
			return K8sContext{Name: "ctx", Namespace: "kubectl-ns"}, nil
		},
		createOrUpdateConfigMapFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "kubectl-ns", namespace)
			return nil
		},
	}
	cmd := newCmd(withConfigLoader(cl), withK8sClient(k8sClient))
	err := cmd.RunConfig(context.Background(), "", "", "")
	require.NoError(t, err)
}

// RunSecret tests

func TestRunSecret_Success(t *testing.T) {
	var loggedInfo, loggedSuccessf, loggedInfof []string
	log := &mockLogger{
		infoFn: func(msg string) {
			loggedInfo = append(loggedInfo, msg)
		},
		infofFn: func(format string, args ...interface{}) {
			loggedInfof = append(loggedInfof, format)
		},
		successfFn: func(format string, args ...interface{}) {
			loggedSuccessf = append(loggedSuccessf, format)
		},
	}

	var createdSecret bool
	var registeredWebhook bool
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			createdSecret = true
			require.Equal(t, "webhook-secrets", name)
			require.Equal(t, "my-namespace", namespace)
			require.Equal(t, "my-cluster", kubeContext)
			require.Contains(t, data["secrets.yaml"], "test-secret")
			return nil
		},
	}
	gh := &mockGitHubClient{
		registerWebhookFn: func(ctx context.Context, owner, repo, webhookURL, secret string) error {
			registeredWebhook = true
			require.Equal(t, "test-owner", owner)
			require.Equal(t, "test-repo", repo)
			require.Contains(t, webhookURL, "ralph.haralovich.org")
			return nil
		},
	}

	cmd := newCmd(withK8sClient(k8sClient), withGitHubClient(gh), withLogger(log))
	err := cmd.RunSecret(context.Background(), "", "")
	require.NoError(t, err)
	require.True(t, createdSecret)
	require.True(t, registeredWebhook)
	require.Contains(t, loggedInfo, "Provisioning webhook-secrets secret...")
}

func TestRunSecret_ConfigLoadFailure(t *testing.T) {
	cl := &mockConfigLoader{
		loadFn: func() (*config.RalphConfig, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(withConfigLoader(cl))
	err := cmd.RunSecret(context.Background(), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRunSecret_KubeContextFailure(t *testing.T) {
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
	err := cmd.RunSecret(context.Background(), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRunSecret_ReadRepoListFailure(t *testing.T) {
	reader := &mockConfigMapReader{
		readWebhookConfigFn: func(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(withConfigMapReader(reader))
	err := cmd.RunSecret(context.Background(), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRunSecret_NoReposFound(t *testing.T) {
	reader := &mockConfigMapReader{
		readWebhookConfigFn: func(ctx context.Context, namespace, kubeContext string) (*webhookconfig.AppConfig, error) {
			return &webhookconfig.AppConfig{Port: 8080}, nil
		},
	}
	cmd := newCmd(withConfigMapReader(reader))
	err := cmd.RunSecret(context.Background(), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no repos found")
}

func TestRunSecret_BuildSecretsFailure(t *testing.T) {
	sb := &mockSecretBuilder{
		buildSecretsFn: func(appCfg *webhookconfig.AppConfig) (*webhookconfig.Secrets, error) {
			return nil, errMockFailure
		},
	}
	cmd := newCmd(withSecretBuilder(sb))
	err := cmd.RunSecret(context.Background(), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRunSecret_WriteSecretsFailure(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			return errMockFailure
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.RunSecret(context.Background(), "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestRunSecret_RegisterWebhookFailure_Warns(t *testing.T) {
	var warned bool
	log := &mockLogger{
		warnfFn: func(format string, args ...interface{}) {
			warned = true
		},
		infofFn: func(format string, args ...interface{}) {},
	}
	gh := &mockGitHubClient{
		registerWebhookFn: func(ctx context.Context, owner, repo, webhookURL, secret string) error {
			return errMockFailure
		},
	}
	cmd := newCmd(withGitHubClient(gh), withLogger(log))
	err := cmd.RunSecret(context.Background(), "", "")
	require.NoError(t, err)
	require.True(t, warned)
}

func TestRunSecret_FlagContextPriority(t *testing.T) {
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
	err := cmd.RunSecret(context.Background(), "flag-ctx", "")
	require.NoError(t, err)
}

func TestRunSecret_ConfigContextPriority(t *testing.T) {
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
	err := cmd.RunSecret(context.Background(), "", "")
	require.NoError(t, err)
}

func TestRunSecret_KubectlContextFallback(t *testing.T) {
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
	err := cmd.RunSecret(context.Background(), "", "")
	require.NoError(t, err)
}

func TestRunSecret_FlagNamespacePriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "flag-ns", namespace)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.RunSecret(context.Background(), "", "flag-ns")
	require.NoError(t, err)
}

func TestRunSecret_ConfigNamespacePriority(t *testing.T) {
	k8sClient := &mockK8sClient{
		createOrUpdateSecretFn: func(ctx context.Context, name, namespace, kubeContext string, data map[string]string) error {
			require.Equal(t, "my-namespace", namespace)
			return nil
		},
	}
	cmd := newCmd(withK8sClient(k8sClient))
	err := cmd.RunSecret(context.Background(), "", "")
	require.NoError(t, err)
}

func TestRunSecret_ConfigPathNamespaceFallback(t *testing.T) {
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
	err := cmd.RunSecret(context.Background(), "", "")
	require.NoError(t, err)
}

func TestRunSecret_DefaultNamespaceFallback(t *testing.T) {
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
	err := cmd.RunSecret(context.Background(), "", "")
	require.NoError(t, err)
}

func TestRunSecret_KubectlNamespaceFallback(t *testing.T) {
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
	err := cmd.RunSecret(context.Background(), "", "")
	require.NoError(t, err)
}
