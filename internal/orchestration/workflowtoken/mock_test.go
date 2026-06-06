package workflowtoken

import (
	"errors"
)

var errMock = errors.New("mock error")

type mockRepoClient struct {
	resolveFunc    func(owner, repo string) (string, string, error)
	resolveCalled  bool
	lastOwner      string
	lastRepo       string
}

func (m *mockRepoClient) Resolve(owner, repo string) (string, string, error) {
	m.resolveCalled = true
	m.lastOwner = owner
	m.lastRepo = repo
	if m.resolveFunc != nil {
		return m.resolveFunc(owner, repo)
	}
	return owner, repo, nil
}

type mockGitHubClient struct {
	generateTokenFunc     func(owner, repo, secretsDir string) (string, error)
	generateTokenCalled   bool
	generateTokenLastOwner string
	generateTokenLastRepo  string
}

func (m *mockGitHubClient) GenerateToken(owner, repo, secretsDir string) (string, error) {
	m.generateTokenCalled = true
	m.generateTokenLastOwner = owner
	m.generateTokenLastRepo = repo
	if m.generateTokenFunc != nil {
		return m.generateTokenFunc(owner, repo, secretsDir)
	}
	return "ghs_mock-token", nil
}

type mockGitClient struct {
	configureAuthFunc   func(token string) error
	configureAuthCalled bool
}

func (m *mockGitClient) ConfigureAuth(token string) error {
	m.configureAuthCalled = true
	if m.configureAuthFunc != nil {
		return m.configureAuthFunc(token)
	}
	return nil
}

var mockRepo *mockRepoClient
var mockGitHub *mockGitHubClient
var mockGit *mockGitClient

type workflowtokenHelper struct{}

type workflowtokenOption func(*WorkflowTokenCmd)

var workflowtoken = &workflowtokenHelper{}

func (h *workflowtokenHelper) withMocks(opts ...workflowtokenOption) *WorkflowTokenCmd {
	mockRepo = &mockRepoClient{}
	mockGitHub = &mockGitHubClient{}
	mockGit = &mockGitClient{}
	cmd := &WorkflowTokenCmd{
		Repo:   mockRepo,
		GitHub: mockGitHub,
		Git:    mockGit,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (h *workflowtokenHelper) withRepo(rc RepoClient) workflowtokenOption {
	return func(cmd *WorkflowTokenCmd) {
		cmd.Repo = rc
		if m, ok := rc.(*mockRepoClient); ok {
			mockRepo = m
		}
	}
}

func (h *workflowtokenHelper) withGitHub(gc GitHubClient) workflowtokenOption {
	return func(cmd *WorkflowTokenCmd) {
		cmd.GitHub = gc
		if m, ok := gc.(*mockGitHubClient); ok {
			mockGitHub = m
		}
	}
}

func (h *workflowtokenHelper) withGit(gc GitClient) workflowtokenOption {
	return func(cmd *WorkflowTokenCmd) {
		cmd.Git = gc
		if m, ok := gc.(*mockGitClient); ok {
			mockGit = m
		}
	}
}

type flagsHelper struct{}

var flags = &flagsHelper{}

func (h *flagsHelper) any() Flags {
	return Flags{
		SecretsDir: "/secrets/github",
	}
}

func (h *flagsHelper) withOwnerAndRepo(owner, repo string) Flags {
	return Flags{
		Owner:      owner,
		Repo:       repo,
		SecretsDir: "/secrets/github",
	}
}

type repoHelper struct{}

var repo = &repoHelper{}

func (h *repoHelper) thatDetectsFromRemote() *mockRepoClient {
	return &mockRepoClient{
		resolveFunc: func(owner, repo string) (string, string, error) {
			if owner == "" && repo == "" {
				return "detected-owner", "detected-repo", nil
			}
			return owner, repo, nil
		},
	}
}

func (h *repoHelper) thatFails() *mockRepoClient {
	return &mockRepoClient{
		resolveFunc: func(_, _ string) (string, string, error) {
			return "", "", errMock
		},
	}
}

func (h *repoHelper) lastResolved() (string, string) {
	if mockRepo != nil {
		return mockRepo.lastOwner, mockRepo.lastRepo
	}
	return "", ""
}

func (h *repoHelper) explicit(owner, repo string) (string, string) {
	return owner, repo
}

type githubHelper struct{}

var github = &githubHelper{}

func (h *githubHelper) generateTokenCalled() bool {
	return mockGitHub != nil && mockGitHub.generateTokenCalled
}

func (h *githubHelper) generateTokenLastArgs() (string, string) {
	if mockGitHub != nil {
		return mockGitHub.generateTokenLastOwner, mockGitHub.generateTokenLastRepo
	}
	return "", ""
}

func (h *githubHelper) thatFailsTokenGeneration() *mockGitHubClient {
	return &mockGitHubClient{
		generateTokenFunc: func(_, _, _ string) (string, error) {
			return "", errMock
		},
	}
}

type gitHelper struct{}

var git = &gitHelper{}

func (h *gitHelper) configureAuthCalled() bool {
	return mockGit != nil && mockGit.configureAuthCalled
}
