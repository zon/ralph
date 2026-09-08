package setup

var errMock = &mockError{"mock error"}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

type mockLocalReadinessClient struct {
	confirmGitFunc         func() error
	confirmGitHubCLIFunc   func() error
	confirmOpenCodeFunc    func() error
	confirmKubectlFunc     func() error
	confirmArgoFunc        func() error
	confirmGitCalled       bool
	confirmGitHubCLICalled bool
	confirmOpenCodeCalled  bool
	confirmKubectlCalled   bool
	confirmArgoCalled      bool
}

func (m *mockLocalReadinessClient) ConfirmGitReady() error {
	m.confirmGitCalled = true
	if m.confirmGitFunc != nil {
		return m.confirmGitFunc()
	}
	return nil
}

func (m *mockLocalReadinessClient) ConfirmGitHubCLIReady() error {
	m.confirmGitHubCLICalled = true
	if m.confirmGitHubCLIFunc != nil {
		return m.confirmGitHubCLIFunc()
	}
	return nil
}

func (m *mockLocalReadinessClient) ConfirmOpenCodeReady() error {
	m.confirmOpenCodeCalled = true
	if m.confirmOpenCodeFunc != nil {
		return m.confirmOpenCodeFunc()
	}
	return nil
}

func (m *mockLocalReadinessClient) ConfirmKubectlReady() error {
	m.confirmKubectlCalled = true
	if m.confirmKubectlFunc != nil {
		return m.confirmKubectlFunc()
	}
	return nil
}

func (m *mockLocalReadinessClient) ConfirmArgoReady() error {
	m.confirmArgoCalled = true
	if m.confirmArgoFunc != nil {
		return m.confirmArgoFunc()
	}
	return nil
}

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

type mockGitHubCredentialsClient struct {
	secretExistsFunc     func(K8sContext) (bool, error)
	validateFunc         func(string) error
	configureFunc        func(K8sContext, string) error
	configureTokenFunc   func(K8sContext, string) error
	ghCliTokenFunc       func() string
	envTokenFunc         func() string
	secretExistsCalled   bool
	validateCalled       bool
	configureCalled      bool
	configureTokenCalled bool
	ghCliTokenCalled     bool
	envTokenCalled       bool
	configuredToken      string
}

func (m *mockGitHubCredentialsClient) SecretExists(k8sCtx K8sContext) (bool, error) {
	m.secretExistsCalled = true
	if m.secretExistsFunc != nil {
		return m.secretExistsFunc(k8sCtx)
	}
	return false, nil
}

func (m *mockGitHubCredentialsClient) Validate(keyPath string) error {
	m.validateCalled = true
	if m.validateFunc != nil {
		return m.validateFunc(keyPath)
	}
	return nil
}

func (m *mockGitHubCredentialsClient) Configure(k8sCtx K8sContext, keyPath string) error {
	m.configureCalled = true
	if m.configureFunc != nil {
		return m.configureFunc(k8sCtx, keyPath)
	}
	return nil
}

func (m *mockGitHubCredentialsClient) ConfigureToken(k8sCtx K8sContext, token string) error {
	m.configureTokenCalled = true
	m.configuredToken = token
	if m.configureTokenFunc != nil {
		return m.configureTokenFunc(k8sCtx, token)
	}
	return nil
}

func (m *mockGitHubCredentialsClient) TokenFromGHCli() string {
	m.ghCliTokenCalled = true
	if m.ghCliTokenFunc != nil {
		return m.ghCliTokenFunc()
	}
	return ""
}

func (m *mockGitHubCredentialsClient) TokenFromEnv() string {
	m.envTokenCalled = true
	if m.envTokenFunc != nil {
		return m.envTokenFunc()
	}
	return ""
}

type mockOpenCodeCredentialsClient struct {
	configureFunc   func(K8sContext) error
	configureCalled bool
}

func (m *mockOpenCodeCredentialsClient) Configure(k8sCtx K8sContext) error {
	m.configureCalled = true
	if m.configureFunc != nil {
		return m.configureFunc(k8sCtx)
	}
	return nil
}

var mockCtx *mockContextClient
var mockGH *mockGitHubCredentialsClient
var mockOC *mockOpenCodeCredentialsClient
var mockReadiness *mockLocalReadinessClient

type setupHelper struct{}

type setupOption func(*SetupCmd)

var setup = &setupHelper{}

func (h *setupHelper) withMocks(opts ...setupOption) *SetupCmd {
	mockCtx = &mockContextClient{}
	mockGH = &mockGitHubCredentialsClient{}
	mockOC = &mockOpenCodeCredentialsClient{}
	mockReadiness = &mockLocalReadinessClient{}
	cmd := &SetupCmd{
		Readiness: mockReadiness,
		Ctx:       mockCtx,
		GitHub:    mockGH,
		OpenCode:  mockOC,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (h *setupHelper) withReadiness(rc LocalReadinessClient) setupOption {
	return func(cmd *SetupCmd) {
		cmd.Readiness = rc
		if m, ok := rc.(*mockLocalReadinessClient); ok {
			mockReadiness = m
		}
	}
}

func (h *setupHelper) withContext(cc ContextClient) setupOption {
	return func(cmd *SetupCmd) {
		cmd.Ctx = cc
		if m, ok := cc.(*mockContextClient); ok {
			mockCtx = m
		}
	}
}

func (h *setupHelper) withGitHub(gc GitHubCredentialsClient) setupOption {
	return func(cmd *SetupCmd) {
		cmd.GitHub = gc
		if m, ok := gc.(*mockGitHubCredentialsClient); ok {
			mockGH = m
		}
	}
}

func (h *setupHelper) withOpenCode(oc OpenCodeCredentialsClient) setupOption {
	return func(cmd *SetupCmd) {
		cmd.OpenCode = oc
		if m, ok := oc.(*mockOpenCodeCredentialsClient); ok {
			mockOC = m
		}
	}
}

type githubHelper struct{}

var github = &githubHelper{}

func (h *githubHelper) validateCalled() bool {
	return mockGH != nil && mockGH.validateCalled
}

func (h *githubHelper) configuredToken() string {
	return mockGH.configuredToken
}

func (h *githubHelper) envTokenCalled() bool {
	return mockGH != nil && mockGH.envTokenCalled
}

func (h *githubHelper) ghCliTokenCalled() bool {
	return mockGH != nil && mockGH.ghCliTokenCalled
}

func (h *githubHelper) configureCalled() bool {
	return mockGH != nil && mockGH.configureCalled
}

func (h *githubHelper) configureTokenCalled() bool {
	return mockGH != nil && mockGH.configureTokenCalled
}

func (h *githubHelper) secretExistsCalled() bool {
	return mockGH != nil && mockGH.secretExistsCalled
}

func (h *githubHelper) thatFailsSecretExists() *mockGitHubCredentialsClient {
	return &mockGitHubCredentialsClient{
		secretExistsFunc: func(K8sContext) (bool, error) { return false, errMock },
	}
}

func (h *githubHelper) thatFailsConfigure() *mockGitHubCredentialsClient {
	return &mockGitHubCredentialsClient{
		validateFunc:  func(string) error { return nil },
		configureFunc: func(K8sContext, string) error { return errMock },
	}
}

func (h *githubHelper) thatFailsValidation() *mockGitHubCredentialsClient {
	return &mockGitHubCredentialsClient{
		validateFunc: func(string) error { return errMock },
	}
}

func (h *githubHelper) thatFailsConfigureToken() *mockGitHubCredentialsClient {
	return &mockGitHubCredentialsClient{
		configureTokenFunc: func(K8sContext, string) error { return errMock },
	}
}

func (h *githubHelper) withExistingSecret() *mockGitHubCredentialsClient {
	return &mockGitHubCredentialsClient{
		secretExistsFunc: func(K8sContext) (bool, error) { return true, nil },
	}
}

func (h *githubHelper) withNoExistingSecret() *mockGitHubCredentialsClient {
	return &mockGitHubCredentialsClient{
		secretExistsFunc: func(K8sContext) (bool, error) { return false, nil },
	}
}

func (h *githubHelper) withGHCliToken(tok string) *mockGitHubCredentialsClient {
	return &mockGitHubCredentialsClient{
		secretExistsFunc: func(K8sContext) (bool, error) { return false, nil },
		ghCliTokenFunc:   func() string { return tok },
		envTokenFunc:     func() string { return "ghp_env_token" },
	}
}

func (h *githubHelper) withEnvToken(tok string) *mockGitHubCredentialsClient {
	return &mockGitHubCredentialsClient{
		secretExistsFunc: func(K8sContext) (bool, error) { return false, nil },
		envTokenFunc:     func() string { return tok },
	}
}

type opencodeHelper struct{}

var opencode = &opencodeHelper{}

func (h *opencodeHelper) configureCalled() bool {
	return mockOC != nil && mockOC.configureCalled
}

func (h *opencodeHelper) thatFails() *mockOpenCodeCredentialsClient {
	return &mockOpenCodeCredentialsClient{
		configureFunc: func(K8sContext) error { return errMock },
	}
}

type ctxHelper struct{}

var ctx = &ctxHelper{}

func (h *ctxHelper) resolveCalled() bool {
	return mockCtx != nil && mockCtx.resolveCalled
}

func (h *ctxHelper) thatFails() *mockContextClient {
	return &mockContextClient{
		resolveFunc: func(string, string) (K8sContext, error) { return K8sContext{}, errMock },
	}
}

func (h *ctxHelper) thatTargetsNoNamespace() *mockContextClient {
	return &mockContextClient{
		resolveFunc: func(string, string) (K8sContext, error) { return K8sContext{}, ErrNoTargetedNamespace },
	}
}

type readinessHelper struct{}

var readiness = &readinessHelper{}

func (h *readinessHelper) confirmGitCalled() bool {
	return mockReadiness != nil && mockReadiness.confirmGitCalled
}

func (h *readinessHelper) confirmGitHubCLICalled() bool {
	return mockReadiness != nil && mockReadiness.confirmGitHubCLICalled
}

func (h *readinessHelper) confirmOpenCodeCalled() bool {
	return mockReadiness != nil && mockReadiness.confirmOpenCodeCalled
}

func (h *readinessHelper) confirmKubectlCalled() bool {
	return mockReadiness != nil && mockReadiness.confirmKubectlCalled
}

func (h *readinessHelper) confirmArgoCalled() bool {
	return mockReadiness != nil && mockReadiness.confirmArgoCalled
}

func (h *readinessHelper) thatFailsGit() *mockLocalReadinessClient {
	return &mockLocalReadinessClient{
		confirmGitFunc: func() error { return errMock },
	}
}

func (h *readinessHelper) thatFailsGitHubCLI() *mockLocalReadinessClient {
	return &mockLocalReadinessClient{
		confirmGitHubCLIFunc: func() error { return errMock },
	}
}

func (h *readinessHelper) thatFailsOpenCode() *mockLocalReadinessClient {
	return &mockLocalReadinessClient{
		confirmOpenCodeFunc: func() error { return errMock },
	}
}

func (h *readinessHelper) thatFailsKubectl() *mockLocalReadinessClient {
	return &mockLocalReadinessClient{
		confirmKubectlFunc: func() error { return errMock },
	}
}

func (h *readinessHelper) thatFailsArgo() *mockLocalReadinessClient {
	return &mockLocalReadinessClient{
		confirmArgoFunc: func() error { return errMock },
	}
}

func (h *readinessHelper) thatRecordsClusterTooling(order *[]string) *mockLocalReadinessClient {
	return &mockLocalReadinessClient{
		confirmKubectlFunc: func() error {
			*order = append(*order, "confirm-kubectl")
			return nil
		},
		confirmArgoFunc: func() error {
			*order = append(*order, "confirm-argo")
			return nil
		},
	}
}

type flagsHelper struct{}

var flags = &flagsHelper{}

func (h *flagsHelper) withKey() Flags {
	return Flags{
		Context:   "test-context",
		Namespace: "test-ns",
		GithubKey: "/path/to/key.pem",
	}
}

func (h *flagsHelper) withToken() Flags {
	return Flags{
		Context:     "test-context",
		Namespace:   "test-ns",
		GithubToken: "ghp_test_token",
	}
}

func (h *flagsHelper) withKeyAndToken() Flags {
	return Flags{
		Context:     "test-context",
		Namespace:   "test-ns",
		GithubKey:   "/path/to/key.pem",
		GithubToken: "ghp_test_token",
	}
}

func (h *flagsHelper) withoutKey() Flags {
	return Flags{
		Context:   "test-context",
		Namespace: "test-ns",
	}
}

func (h *flagsHelper) withoutNamespace() Flags {
	return Flags{
		Context: "test-context",
	}
}
