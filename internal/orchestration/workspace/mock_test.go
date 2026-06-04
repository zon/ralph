package workspace

// mockGitHubClient
type mockGitHubClient struct {
	configureAuthFunc   func(repo string) error
	configureAuthCalled bool
}

func (m *mockGitHubClient) ConfigureAuth(repo string) error {
	m.configureAuthCalled = true
	if m.configureAuthFunc != nil {
		return m.configureAuthFunc(repo)
	}
	return nil
}

// mockWorkspaceClient
type mockWorkspaceClient struct {
	setupCredentialsFunc   func() error
	setupSymlinksFunc      func() error
	setupCredentialsCalled bool
	setupSymlinksCalled    bool
}

func (m *mockWorkspaceClient) SetupCredentials() error {
	m.setupCredentialsCalled = true
	if m.setupCredentialsFunc != nil {
		return m.setupCredentialsFunc()
	}
	return nil
}

func (m *mockWorkspaceClient) SetupSymlinks() error {
	m.setupSymlinksCalled = true
	if m.setupSymlinksFunc != nil {
		return m.setupSymlinksFunc()
	}
	return nil
}

// mockGitClient
type mockGitClient struct {
	configureUserFunc        func(name, email string)
	cloneFunc                func(branch string) error
	remoteBranchExistsFunc   func(branch string) (bool, error)
	fetchAndCheckoutFunc     func(branch string) error
	createAndCheckoutFunc    func(branch string) error

	configureUserCalled        bool
	cloneCalled                bool
	remoteBranchExistsCalled   bool
	fetchAndCheckoutCalled     bool
	createAndCheckoutCalled    bool
}

func (m *mockGitClient) ConfigureUser(name, email string) {
	m.configureUserCalled = true
	if m.configureUserFunc != nil {
		m.configureUserFunc(name, email)
	}
}

func (m *mockGitClient) Clone(branch string) error {
	m.cloneCalled = true
	if m.cloneFunc != nil {
		return m.cloneFunc(branch)
	}
	return nil
}

func (m *mockGitClient) RemoteBranchExists(branch string) (bool, error) {
	m.remoteBranchExistsCalled = true
	if m.remoteBranchExistsFunc != nil {
		return m.remoteBranchExistsFunc(branch)
	}
	return false, nil
}

func (m *mockGitClient) FetchAndCheckout(branch string) error {
	m.fetchAndCheckoutCalled = true
	if m.fetchAndCheckoutFunc != nil {
		return m.fetchAndCheckoutFunc(branch)
	}
	return nil
}

func (m *mockGitClient) CreateAndCheckout(branch string) error {
	m.createAndCheckoutCalled = true
	if m.createAndCheckoutFunc != nil {
		return m.createAndCheckoutFunc(branch)
	}
	return nil
}

// option types
type workspaceOption func(*WorkspaceSetup)

func withGitHub(gc GitHubClient) workspaceOption {
	return func(w *WorkspaceSetup) {
		w.github = gc
	}
}

func withWorkspace(wc WorkspaceClient) workspaceOption {
	return func(w *WorkspaceSetup) {
		w.workspace = wc
	}
}

func withGit(gc GitClient) workspaceOption {
	return func(w *WorkspaceSetup) {
		w.git = gc
	}
}

func withMocks(opts ...workspaceOption) *WorkspaceSetup {
	w := &WorkspaceSetup{
		github:    &mockGitHubClient{},
		workspace: &mockWorkspaceClient{},
		git:       &mockGitClient{},
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// flag helpers
func flagsAny() WorkspaceFlags {
	return WorkspaceFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
		Symlinks:    true,
	}
}

func flagsWithNoTargetBranch() WorkspaceFlags {
	return WorkspaceFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
	}
}

func flagsWithTargetBranch(branch string) WorkspaceFlags {
	return WorkspaceFlags{
		Repo:         "owner/repo",
		CloneBranch:  "main",
		TargetBranch: branch,
		Symlinks:     true,
	}
}

func (f WorkspaceFlags) withCreateBranch() WorkspaceFlags {
	f.CreateBranch = true
	return f
}

func flagsWithSymlinksDisabled() WorkspaceFlags {
	return WorkspaceFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
	}
}

func flagsWithSymlinksEnabled() WorkspaceFlags {
	return WorkspaceFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
		Symlinks:    true,
	}
}

// mock builders for GitHub
func thatFailsAuth() *mockGitHubClient {
	return &mockGitHubClient{
		configureAuthFunc: func(repo string) error {
			return errMock
		},
	}
}

// mock builders for workspace
func thatFailsCredentials() *mockWorkspaceClient {
	return &mockWorkspaceClient{
		setupCredentialsFunc: func() error {
			return errMock
		},
	}
}

// mock builders for git
func thatFailsClone() *mockGitClient {
	return &mockGitClient{
		cloneFunc: func(branch string) error {
			return errMock
		},
	}
}

func thatReportsRemoteBranchExists() *mockGitClient {
	return &mockGitClient{
		remoteBranchExistsFunc: func(branch string) (bool, error) {
			return true, nil
		},
	}
}

func thatReportsRemoteBranchAbsent() *mockGitClient {
	return &mockGitClient{
		remoteBranchExistsFunc: func(branch string) (bool, error) {
			return false, nil
		},
	}
}

// accessor helpers
func credentialsSetUp(cmd *WorkspaceSetup) bool {
	if m, ok := cmd.workspace.(*mockWorkspaceClient); ok {
		return m.setupCredentialsCalled
	}
	return false
}

func symlinksSetUp(cmd *WorkspaceSetup) bool {
	if m, ok := cmd.workspace.(*mockWorkspaceClient); ok {
		return m.setupSymlinksCalled
	}
	return false
}

func cloned(cmd *WorkspaceSetup) bool {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.cloneCalled
	}
	return false
}

func checkoutCalled(cmd *WorkspaceSetup) bool {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.fetchAndCheckoutCalled || m.createAndCheckoutCalled
	}
	return false
}

func fetchAndCheckoutCalled(cmd *WorkspaceSetup) bool {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.fetchAndCheckoutCalled
	}
	return false
}

func createAndCheckoutCalled(cmd *WorkspaceSetup) bool {
	if m, ok := cmd.git.(*mockGitClient); ok {
		return m.createAndCheckoutCalled
	}
	return false
}

var errMock = &mockError{"mock error"}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
