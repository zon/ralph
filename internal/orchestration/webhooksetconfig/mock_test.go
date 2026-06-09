package webhooksetconfig

import "github.com/zon/ralph/internal/webhookconfig"

var errMock = &mockError{"mock error"}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

type mockContextClient struct {
	resolveFunc   func(string, string) (K8sContext, error)
	resolveCalled bool
}

func (m *mockContextClient) Resolve(flagContext, flagNamespace string) (K8sContext, error) {
	m.resolveCalled = true
	if m.resolveFunc != nil {
		return m.resolveFunc(flagContext, flagNamespace)
	}
	return K8sContext{Name: "test-context", Namespace: "test-ns"}, nil
}

type mockConfigClient struct {
	buildFunc     func(K8sContext, string) webhookconfig.AppConfig
	writeFunc     func(K8sContext, webhookconfig.AppConfig) error
	readFunc      func(K8sContext) (webhookconfig.AppConfig, error)
	buildCalled   bool
	writeCalled   bool
	readCalled    bool
}

func (m *mockConfigClient) Build(k8sCtx K8sContext, configPath string) webhookconfig.AppConfig {
	m.buildCalled = true
	if m.buildFunc != nil {
		return m.buildFunc(k8sCtx, configPath)
	}
	return webhookconfig.AppConfig{Port: 8080}
}

func (m *mockConfigClient) Write(k8sCtx K8sContext, cfg webhookconfig.AppConfig) error {
	m.writeCalled = true
	if m.writeFunc != nil {
		return m.writeFunc(k8sCtx, cfg)
	}
	return nil
}

func (m *mockConfigClient) Read(k8sCtx K8sContext) (webhookconfig.AppConfig, error) {
	m.readCalled = true
	if m.readFunc != nil {
		return m.readFunc(k8sCtx)
	}
	return webhookconfig.AppConfig{Port: 8080}, nil
}

type mockSecretsClient struct {
	generateFunc   func(webhookconfig.AppConfig) (WebhookSecrets, error)
	writeFunc      func(K8sContext, WebhookSecrets) error
	generateCalled bool
	writeCalled    bool
}

func (m *mockSecretsClient) Generate(cfg webhookconfig.AppConfig) (WebhookSecrets, error) {
	m.generateCalled = true
	if m.generateFunc != nil {
		return m.generateFunc(cfg)
	}
	return WebhookSecrets{}, nil
}

func (m *mockSecretsClient) Write(k8sCtx K8sContext, secrets WebhookSecrets) error {
	m.writeCalled = true
	if m.writeFunc != nil {
		return m.writeFunc(k8sCtx, secrets)
	}
	return nil
}

type mockGitHubClient struct {
	registerWebhooksFunc func(WebhookSecrets)
	registerCalled       bool
}

func (m *mockGitHubClient) RegisterWebhooks(secrets WebhookSecrets) {
	m.registerCalled = true
	if m.registerWebhooksFunc != nil {
		m.registerWebhooksFunc(secrets)
	}
}

var mockCtx *mockContextClient
var mockCfg *mockConfigClient
var mockSec *mockSecretsClient
var mockGH *mockGitHubClient

type webhooksetconfigHelper struct{}

type webhooksetconfigOption func(*SetConfigCmd)

var webhooksetconfig = &webhooksetconfigHelper{}

func (h *webhooksetconfigHelper) withMocks(opts ...webhooksetconfigOption) *SetConfigCmd {
	mockCtx = &mockContextClient{}
	mockCfg = &mockConfigClient{}
	mockSec = &mockSecretsClient{}
	mockGH = &mockGitHubClient{}
	cmd := &SetConfigCmd{
		Ctx:     mockCtx,
		Config:  mockCfg,
		Secrets: mockSec,
		GitHub:  mockGH,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (h *webhooksetconfigHelper) withContext(cc ContextClient) webhooksetconfigOption {
	return func(cmd *SetConfigCmd) {
		cmd.Ctx = cc
		if m, ok := cc.(*mockContextClient); ok {
			mockCtx = m
		}
	}
}

func (h *webhooksetconfigHelper) withConfig(cc ConfigClient) webhooksetconfigOption {
	return func(cmd *SetConfigCmd) {
		cmd.Config = cc
		if m, ok := cc.(*mockConfigClient); ok {
			mockCfg = m
		}
	}
}

func (h *webhooksetconfigHelper) withSecrets(sc SecretsClient) webhooksetconfigOption {
	return func(cmd *SetConfigCmd) {
		cmd.Secrets = sc
		if m, ok := sc.(*mockSecretsClient); ok {
			mockSec = m
		}
	}
}

func (h *webhooksetconfigHelper) withGitHub(gc GitHubClient) webhooksetconfigOption {
	return func(cmd *SetConfigCmd) {
		cmd.GitHub = gc
		if m, ok := gc.(*mockGitHubClient); ok {
			mockGH = m
		}
	}
}

type configHelper struct{}

var config = &configHelper{}

func (h *configHelper) writeCalled() bool {
	return mockCfg != nil && mockCfg.writeCalled
}

func (h *configHelper) thatFailsWrite() *mockConfigClient {
	return &mockConfigClient{
		writeFunc: func(K8sContext, webhookconfig.AppConfig) error { return errMock },
	}
}

type secretsHelper struct{}

var secrets = &secretsHelper{}

func (h *secretsHelper) generateCalled() bool {
	return mockSec != nil && mockSec.generateCalled
}

func (h *secretsHelper) writeCalled() bool {
	return mockSec != nil && mockSec.writeCalled
}

type githubHelper struct{}

var github = &githubHelper{}

func (h *githubHelper) registerCalled() bool {
	return mockGH != nil && mockGH.registerCalled
}

type ctxHelper struct{}

var ctx = &ctxHelper{}

func (h *ctxHelper) thatFails() *mockContextClient {
	return &mockContextClient{
		resolveFunc: func(string, string) (K8sContext, error) { return K8sContext{}, errMock },
	}
}

type flagsHelper struct{}

var flags = &flagsHelper{}

func (h *flagsHelper) any() Flags {
	return Flags{
		Context:    "test-context",
		Namespace:  "test-ns",
		ConfigPath: "/path/to/config.yaml",
	}
}
