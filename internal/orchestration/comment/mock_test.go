package comment

import (
	ralphcfg "github.com/zon/ralph/internal/config"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
	ralphsvc "github.com/zon/ralph/internal/services"
)

var errMock = &mockError{"mock error"}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

type mockWorkspaceSetupClient struct {
	setupFunc   func(wksp.WorkspaceFlags) error
	setupCalled bool
}

func (m *mockWorkspaceSetupClient) Setup(flags wksp.WorkspaceFlags) error {
	m.setupCalled = true
	if m.setupFunc != nil {
		return m.setupFunc(flags)
	}
	return nil
}

type mockConfigClient struct {
	loadOptionalFunc   func() (*ralphcfg.RalphConfig, error)
	loadOptionalCalled bool
}

func (m *mockConfigClient) LoadOptional() (*ralphcfg.RalphConfig, error) {
	m.loadOptionalCalled = true
	if m.loadOptionalFunc != nil {
		return m.loadOptionalFunc()
	}
	return ralphcfg.Any(), nil
}

type mockAIClient struct {
	renderCommentPromptFunc    func(CommentContext, string) (string, error)
	runAgentFunc               func(string) error
	generateChangelogFunc      func() error
	generateCommentReplyFunc   func(CommentContext, bool) (string, error)
	renderCommentPromptCalled  bool
	runAgentCalled             bool
	generateChangelogCalled    bool
	generateCommentReplyCalled bool
}

func (m *mockAIClient) RenderCommentPrompt(ctx CommentContext, instructionsFile string) (string, error) {
	m.renderCommentPromptCalled = true
	if m.renderCommentPromptFunc != nil {
		return m.renderCommentPromptFunc(ctx, instructionsFile)
	}
	return "prompt", nil
}

func (m *mockAIClient) RunAgent(prompt string) error {
	m.runAgentCalled = true
	if mockServices != nil && mockServices.started {
		servicesStartedBeforeAgent = true
	}
	if m.runAgentFunc != nil {
		return m.runAgentFunc(prompt)
	}
	return nil
}

func (m *mockAIClient) GenerateChangelog() error {
	m.generateChangelogCalled = true
	if m.generateChangelogFunc != nil {
		return m.generateChangelogFunc()
	}
	return nil
}

func (m *mockAIClient) GenerateCommentReply(ctx CommentContext, pushed bool) (string, error) {
	m.generateCommentReplyCalled = true
	if m.generateCommentReplyFunc != nil {
		return m.generateCommentReplyFunc(ctx, pushed)
	}
	return "reply", nil
}

type mockServicesClient struct {
	startFunc func(*ralphcfg.RalphConfig) (*ralphsvc.Manager, error)
	stopFunc  func(*ralphsvc.Manager)
	started   bool
	startCnt  int
	stopCnt   int
}

func (m *mockServicesClient) Start(cfg *ralphcfg.RalphConfig) (*ralphsvc.Manager, error) {
	m.started = true
	m.startCnt++
	if m.startFunc != nil {
		return m.startFunc(cfg)
	}
	return &ralphsvc.Manager{}, nil
}

func (m *mockServicesClient) Stop(svc *ralphsvc.Manager) {
	m.stopCnt++
	if m.stopFunc != nil {
		m.stopFunc(svc)
	}
}

type mockGitClient struct {
	hasChangesFunc             func() bool
	reportExistsFunc           func() bool
	commitAndPushFromReportFunc func() error
	hasChangesCalled           bool
	reportExistsCalled         bool
	committedAndPushed         bool
}

func (m *mockGitClient) HasChanges() bool {
	m.hasChangesCalled = true
	if m.hasChangesFunc != nil {
		return m.hasChangesFunc()
	}
	return false
}

func (m *mockGitClient) ReportExists() bool {
	m.reportExistsCalled = true
	if m.reportExistsFunc != nil {
		return m.reportExistsFunc()
	}
	return false
}

func (m *mockGitClient) CommitAndPushFromReport() error {
	m.committedAndPushed = true
	if m.commitAndPushFromReportFunc != nil {
		return m.commitAndPushFromReportFunc()
	}
	return nil
}

type mockGitHubClient struct {
	postCommentFunc func(int, string) error
	commentPosted   bool
}

func (m *mockGitHubClient) PostComment(prNumber int, body string) error {
	m.commentPosted = true
	if m.postCommentFunc != nil {
		return m.postCommentFunc(prNumber, body)
	}
	return nil
}

var mockWksp *mockWorkspaceSetupClient
var mockCfg *mockConfigClient
var mockAI *mockAIClient
var mockServices *mockServicesClient
var mockGit *mockGitClient
var mockGH *mockGitHubClient
var servicesStartedBeforeAgent bool

type commentHelper struct{}

type commentOption func(*WorkflowCommentCmd)

var comment = &commentHelper{}

func (h *commentHelper) withMocks(opts ...commentOption) *WorkflowCommentCmd {
	mockWksp = &mockWorkspaceSetupClient{}
	mockCfg = &mockConfigClient{}
	mockAI = &mockAIClient{}
	mockServices = &mockServicesClient{}
	mockGit = &mockGitClient{}
	mockGH = &mockGitHubClient{}
	servicesStartedBeforeAgent = false
	cmd := &WorkflowCommentCmd{
		workspace: mockWksp,
		config:    mockCfg,
		ai:        mockAI,
		services:  mockServices,
		git:       mockGit,
		github:    mockGH,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (h *commentHelper) withWorkspace(wc WorkspaceSetupClient) commentOption {
	return func(cmd *WorkflowCommentCmd) {
		cmd.workspace = wc
		if m, ok := wc.(*mockWorkspaceSetupClient); ok {
			mockWksp = m
		}
	}
}

func (h *commentHelper) withGit(gc GitClient) commentOption {
	return func(cmd *WorkflowCommentCmd) {
		cmd.git = gc
		if m, ok := gc.(*mockGitClient); ok {
			mockGit = m
		}
	}
}

type workspaceHelper struct{}

var workspace = &workspaceHelper{}

func (h *workspaceHelper) thatFailsSetup() *mockWorkspaceSetupClient {
	return &mockWorkspaceSetupClient{
		setupFunc: func(wksp.WorkspaceFlags) error { return errMock },
	}
}

func (h *workspaceHelper) setupCalled() bool {
	return mockWksp != nil && mockWksp.setupCalled
}

type configHelper struct{}

var config = &configHelper{}

func (h *configHelper) loadCalled() bool {
	return mockCfg != nil && mockCfg.loadOptionalCalled
}

type gitHelper struct{}

var git = &gitHelper{}

func (h *gitHelper) withChangesAndReport() *mockGitClient {
	return &mockGitClient{
		hasChangesFunc:   func() bool { return true },
		reportExistsFunc: func() bool { return true },
	}
}

func (h *gitHelper) withNoChanges() *mockGitClient {
	return &mockGitClient{
		hasChangesFunc: func() bool { return false },
	}
}

func (h *gitHelper) committedAndPushed() bool {
	return mockGit != nil && mockGit.committedAndPushed
}

type aiHelper struct{}

var ai = &aiHelper{}

type servicesHelper struct{}

var svcs = &servicesHelper{}

func (h *servicesHelper) startCount() int {
	if mockServices == nil {
		return 0
	}
	return mockServices.startCnt
}

func (h *servicesHelper) stopCount() int {
	if mockServices == nil {
		return 0
	}
	return mockServices.stopCnt
}

func (h *servicesHelper) startedBeforeAgent() bool {
	return servicesStartedBeforeAgent
}

type githubHelper struct{}

var github = &githubHelper{}

func (h *githubHelper) commentPosted() bool {
	return mockGH != nil && mockGH.commentPosted
}

type flagsHelper struct{}

var flags = &flagsHelper{}

func (h *flagsHelper) any() WorkflowCommentFlags {
	return WorkflowCommentFlags{
		Repo:          "owner/repo",
		CloneBranch:   "main",
		ProjectBranch: "feature/test",
		BotName:       "ralph",
		BotEmail:      "ralph@example.com",
		CommentBody:   "Please review this PR",
		PRNumber:      42,
		RepoOwner:     "owner",
		RepoName:      "repo",
	}
}

func (h *flagsHelper) withNoCommentBody() WorkflowCommentFlags {
	f := h.any()
	f.CommentBody = ""
	return f
}

func (h *flagsHelper) withNoServices() WorkflowCommentFlags {
	f := h.any()
	f.NoServices = true
	return f
}
