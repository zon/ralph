package workflowrun

import (
	ralphcfg "github.com/zon/ralph/internal/config"
	wksp "github.com/zon/ralph/internal/orchestration/workspace"
	ralphproj "github.com/zon/ralph/internal/project"
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

type mockGitClient struct {
	fetchBranchFunc func(string) error
	needsMergeFunc  func(string) (bool, error)
	mergeFunc       func(string) error
	abortMergeFunc  func()

	fetchBranchCalled bool
	needsMergeCalled  bool
	mergeCalled       bool
	abortMergeCalled  bool
}

func (m *mockGitClient) FetchBranch(branch string) error {
	m.fetchBranchCalled = true
	if m.fetchBranchFunc != nil {
		return m.fetchBranchFunc(branch)
	}
	return nil
}

func (m *mockGitClient) NeedsMerge(branch string) (bool, error) {
	m.needsMergeCalled = true
	if m.needsMergeFunc != nil {
		return m.needsMergeFunc(branch)
	}
	return false, nil
}

func (m *mockGitClient) Merge(branch string) error {
	m.mergeCalled = true
	if m.mergeFunc != nil {
		return m.mergeFunc(branch)
	}
	return nil
}

func (m *mockGitClient) AbortMerge() {
	m.abortMergeCalled = true
	if m.abortMergeFunc != nil {
		m.abortMergeFunc()
	}
}

type mockAIClient struct {
	resolveMergeConflictsFunc   func(string, string) error
	resolveMergeConflictsCalled bool
}

func (m *mockAIClient) ResolveMergeConflicts(baseBranch, projectBranch string) error {
	m.resolveMergeConflictsCalled = true
	if m.resolveMergeConflictsFunc != nil {
		return m.resolveMergeConflictsFunc(baseBranch, projectBranch)
	}
	return nil
}

type mockRunnerClient struct {
	runLocalFunc   func(*ralphproj.Project, *ralphcfg.RalphConfig) error
	runLocalCalled bool
}

func (m *mockRunnerClient) RunLocal(proj *ralphproj.Project, cfg *ralphcfg.RalphConfig) error {
	m.runLocalCalled = true
	if m.runLocalFunc != nil {
		return m.runLocalFunc(proj, cfg)
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

type mockProjectClient struct {
	loadFunc   func(string) (*ralphproj.Project, error)
	loadCalled bool
}

func (m *mockProjectClient) Load(path string) (*ralphproj.Project, error) {
	m.loadCalled = true
	if m.loadFunc != nil {
		return m.loadFunc(path)
	}
	return &ralphproj.Project{Slug: "test-project"}, nil
}

type mockDebugClient struct {
	setupFunc   func(string) error
	setupCalled bool
}

func (m *mockDebugClient) Setup(branch string) error {
	m.setupCalled = true
	if m.setupFunc != nil {
		return m.setupFunc(branch)
	}
	return nil
}

type mockOutputClient struct {
	warnfFunc   func(string, ...any)
	warnfCalled bool
}

func (m *mockOutputClient) Warnf(format string, a ...any) {
	m.warnfCalled = true
	if m.warnfFunc != nil {
		m.warnfFunc(format, a...)
	}
}

var mockWksp *mockWorkspaceSetupClient
var mockGit *mockGitClient
var mockAI *mockAIClient
var mockRunner *mockRunnerClient
var mockCfg *mockConfigClient
var mockProj *mockProjectClient
var mockDebug *mockDebugClient
var mockOutput *mockOutputClient

type runHelper struct{}

type runOption func(*WorkflowRunCmd)

var run = &runHelper{}

func (r *runHelper) withRunner(rc RunnerClient) runOption {
	return func(cmd *WorkflowRunCmd) {
		cmd.runner = rc
		if m, ok := rc.(*mockRunnerClient); ok {
			mockRunner = m
		}
	}
}

func (r *runHelper) withMocks(opts ...runOption) *WorkflowRunCmd {
	mockWksp = &mockWorkspaceSetupClient{}
	mockGit = &mockGitClient{}
	mockAI = &mockAIClient{}
	mockRunner = &mockRunnerClient{}
	mockCfg = &mockConfigClient{}
	mockProj = &mockProjectClient{}
	mockDebug = &mockDebugClient{}
	mockOutput = &mockOutputClient{}
	cmd := &WorkflowRunCmd{
		workspace: mockWksp,
		git:       mockGit,
		ai:        mockAI,
		runner:    mockRunner,
		config:    mockCfg,
		project:   mockProj,
		debug:     mockDebug,
		output:    mockOutput,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (r *runHelper) withWorkspace(wc WorkspaceSetupClient) runOption {
	return func(cmd *WorkflowRunCmd) {
		cmd.workspace = wc
		if m, ok := wc.(*mockWorkspaceSetupClient); ok {
			mockWksp = m
		}
	}
}

func (r *runHelper) withGit(gc GitClient) runOption {
	return func(cmd *WorkflowRunCmd) {
		cmd.git = gc
		if m, ok := gc.(*mockGitClient); ok {
			mockGit = m
		}
	}
}

func (r *runHelper) withAI(ac AIClient) runOption {
	return func(cmd *WorkflowRunCmd) {
		cmd.ai = ac
		if m, ok := ac.(*mockAIClient); ok {
			mockAI = m
		}
	}
}

func (r *runHelper) withConfig(cc ConfigClient) runOption {
	return func(cmd *WorkflowRunCmd) {
		cmd.config = cc
		if m, ok := cc.(*mockConfigClient); ok {
			mockCfg = m
		}
	}
}

func (r *runHelper) withProject(pc ProjectClient) runOption {
	return func(cmd *WorkflowRunCmd) {
		cmd.project = pc
		if m, ok := pc.(*mockProjectClient); ok {
			mockProj = m
		}
	}
}

func (r *runHelper) withDebug(dc DebugClient) runOption {
	return func(cmd *WorkflowRunCmd) {
		cmd.debug = dc
		if m, ok := dc.(*mockDebugClient); ok {
			mockDebug = m
		}
	}
}

func (r *runHelper) withOutput(oc OutputClient) runOption {
	return func(cmd *WorkflowRunCmd) {
		cmd.output = oc
		if m, ok := oc.(*mockOutputClient); ok {
			mockOutput = m
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

func (h *configHelper) thatReportsMissing() *mockConfigClient {
	return &mockConfigClient{}
}

func (h *configHelper) thatFailsParsing() *mockConfigClient {
	return &mockConfigClient{
		loadOptionalFunc: func() (*ralphcfg.RalphConfig, error) {
			return nil, errMock
		},
	}
}

func (h *configHelper) loadCalled() bool {
	return mockCfg != nil && mockCfg.loadOptionalCalled
}

type gitHelper struct{}

var git = &gitHelper{}

func (h *gitHelper) thatFailsFetch() *mockGitClient {
	return &mockGitClient{
		fetchBranchFunc: func(string) error { return errMock },
	}
}

func (h *gitHelper) thatReportsUpToDate() *mockGitClient {
	return &mockGitClient{
		needsMergeFunc: func(string) (bool, error) { return false, nil },
	}
}

func (h *gitHelper) thatNeedsMerge() *mockGitClient {
	return &mockGitClient{
		needsMergeFunc: func(string) (bool, error) { return true, nil },
	}
}

func (m *mockGitClient) thatProducesConflicts() *mockGitClient {
	m.mergeFunc = func(string) error { return errMock }
	return m
}

func (h *gitHelper) fetchCalled() bool {
	return mockGit != nil && mockGit.fetchBranchCalled
}

func (h *gitHelper) mergeCalled() bool {
	return mockGit != nil && mockGit.mergeCalled
}

func (h *gitHelper) mergeAborted() bool {
	return mockGit != nil && mockGit.abortMergeCalled
}

type aiHelper struct{}

var ai = &aiHelper{}

func (h *aiHelper) conflictsResolved() bool {
	return mockAI != nil && mockAI.resolveMergeConflictsCalled
}

type runnerHelper struct{}

var runner = &runnerHelper{}

func (h *runnerHelper) runLocalCalled() bool {
	return mockRunner != nil && mockRunner.runLocalCalled
}

type projectHelper struct{}

var project = &projectHelper{}

func (h *projectHelper) thatFailsLoad() *mockProjectClient {
	return &mockProjectClient{
		loadFunc: func(string) (*ralphproj.Project, error) {
			return nil, errMock
		},
	}
}

func (h *projectHelper) loadCalled() bool {
	return mockProj != nil && mockProj.loadCalled
}

type outputHelper struct{}

var output = &outputHelper{}

func (h *outputHelper) warnfCalled() bool {
	return mockOutput != nil && mockOutput.warnfCalled
}

type debugHelper struct{}

var debug = &debugHelper{}

func (h *debugHelper) thatFailsSetup() *mockDebugClient {
	return &mockDebugClient{
		setupFunc: func(string) error { return errMock },
	}
}

type flagsHelper struct{}

var flags = &flagsHelper{}

func (h *flagsHelper) any() WorkflowRunFlags {
	return WorkflowRunFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
		BaseBranch:  "main",
		ProjectPath: "projects/test.yaml",
		BotName:     "ralph",
		BotEmail:    "ralph@example.com",
	}
}

func (h *flagsHelper) withExtraIterations(n int) WorkflowRunFlags {
	f := h.any()
	f.ExtraIterations = n
	return f
}

func (h *flagsHelper) withNoProjectPath() WorkflowRunFlags {
	f := h.any()
	f.ProjectPath = ""
	return f
}

func (h *flagsHelper) withDebugBranch(branch string) WorkflowRunFlags {
	f := h.any()
	f.Debug = branch
	return f
}
