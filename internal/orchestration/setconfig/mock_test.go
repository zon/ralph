package setconfig

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

type mockGitHubCredentialsClient struct {
	secretExistsFunc   func(K8sContext) (bool, error)
	validateFunc       func(string) error
	configureFunc      func(K8sContext, string) error
	secretExistsCalled bool
	validateCalled     bool
	configureCalled    bool
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

type setconfigHelper struct{}

type setconfigOption func(*SetConfigCmd)

var setconfig = &setconfigHelper{}

func (h *setconfigHelper) withMocks(opts ...setconfigOption) *SetConfigCmd {
	mockCtx = &mockContextClient{}
	mockGH = &mockGitHubCredentialsClient{}
	mockOC = &mockOpenCodeCredentialsClient{}
	cmd := &SetConfigCmd{
		Ctx:      mockCtx,
		GitHub:   mockGH,
		OpenCode: mockOC,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (h *setconfigHelper) withContext(cc ContextClient) setconfigOption {
	return func(cmd *SetConfigCmd) {
		cmd.Ctx = cc
		if m, ok := cc.(*mockContextClient); ok {
			mockCtx = m
		}
	}
}

func (h *setconfigHelper) withGitHub(gc GitHubCredentialsClient) setconfigOption {
	return func(cmd *SetConfigCmd) {
		cmd.GitHub = gc
		if m, ok := gc.(*mockGitHubCredentialsClient); ok {
			mockGH = m
		}
	}
}

func (h *setconfigHelper) withOpenCode(oc OpenCodeCredentialsClient) setconfigOption {
	return func(cmd *SetConfigCmd) {
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

func (h *githubHelper) configureCalled() bool {
	return mockGH != nil && mockGH.configureCalled
}

func (h *githubHelper) thatFailsValidation() *mockGitHubCredentialsClient {
	return &mockGitHubCredentialsClient{
		validateFunc: func(string) error { return errMock },
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

func (h *ctxHelper) thatFails() *mockContextClient {
	return &mockContextClient{
		resolveFunc: func(string, string) (K8sContext, error) { return K8sContext{}, errMock },
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

func (h *flagsHelper) withoutKey() Flags {
	return Flags{
		Context:   "test-context",
		Namespace: "test-ns",
	}
}
