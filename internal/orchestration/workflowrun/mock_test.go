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
	resolveFunc func(string, string) (*ralphproj.Project, error)
	lastQuery   string
}

func (m *mockProjectClient) Resolve(path, query string) (*ralphproj.Project, error) {
	m.lastQuery = query
	if m.resolveFunc != nil {
		return m.resolveFunc(path, query)
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

var mockWksp *mockWorkspaceSetupClient
var mockRunner *mockRunnerClient
var mockCfg *mockConfigClient
var mockProj *mockProjectClient
var mockDebug *mockDebugClient

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
	mockRunner = &mockRunnerClient{}
	mockCfg = &mockConfigClient{}
	mockProj = &mockProjectClient{}
	mockDebug = &mockDebugClient{}
	cmd := &WorkflowRunCmd{
		workspace: mockWksp,
		runner:    mockRunner,
		config:    mockCfg,
		project:   mockProj,
		debug:     mockDebug,
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

type runnerHelper struct{}

var runner = &runnerHelper{}

func (h *runnerHelper) runLocalCalled() bool {
	return mockRunner != nil && mockRunner.runLocalCalled
}

type projectHelper struct{}

var project = &projectHelper{}

func (h *projectHelper) thatFailsResolve() *mockProjectClient {
	return &mockProjectClient{
		resolveFunc: func(string, string) (*ralphproj.Project, error) {
			return nil, errMock
		},
	}
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

func (h *flagsHelper) withItems(query string) WorkflowRunFlags {
	f := h.any()
	f.Items = query
	return f
}
