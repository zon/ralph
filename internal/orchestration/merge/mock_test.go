package merge

import (
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
	commitAndPushFunc   func(string) error
	commitAndPushCalled bool
	commitAndPushMsg    string
}

func (m *mockGitClient) CommitAndPush(message string) error {
	m.commitAndPushCalled = true
	m.commitAndPushMsg = message
	if m.commitAndPushFunc != nil {
		return m.commitAndPushFunc(message)
	}
	return nil
}

type mockGitHubClient struct {
	waitForHeadSyncFunc   func(string) error
	mergePRFunc           func(int) error
	waitForHeadSyncCalled bool
	mergePRCalled         bool
}

func (m *mockGitHubClient) WaitForHeadSync(prBranch string) error {
	m.waitForHeadSyncCalled = true
	if m.waitForHeadSyncFunc != nil {
		return m.waitForHeadSyncFunc(prBranch)
	}
	return nil
}

func (m *mockGitHubClient) MergePR(prNumber int) error {
	m.mergePRCalled = true
	if m.mergePRFunc != nil {
		return m.mergePRFunc(prNumber)
	}
	return nil
}

type mockProjectClient struct {
	loadAllFunc        func() ([]*ralphproj.Project, error)
	filterPassingFunc  func([]*ralphproj.Project) []*ralphproj.Project
	deleteAllFunc      func([]*ralphproj.Project) error
	loadAllCalled      bool
	filterPassingCalled bool
	deleteAllCalled    bool
}

func (m *mockProjectClient) LoadAll() ([]*ralphproj.Project, error) {
	m.loadAllCalled = true
	if m.loadAllFunc != nil {
		return m.loadAllFunc()
	}
	return nil, nil
}

func (m *mockProjectClient) FilterPassing(projects []*ralphproj.Project) []*ralphproj.Project {
	m.filterPassingCalled = true
	if m.filterPassingFunc != nil {
		return m.filterPassingFunc(projects)
	}
	return nil
}

func (m *mockProjectClient) DeleteAll(projects []*ralphproj.Project) error {
	m.deleteAllCalled = true
	if m.deleteAllFunc != nil {
		return m.deleteAllFunc(projects)
	}
	return nil
}

var mockWksp *mockWorkspaceSetupClient
var mockGit *mockGitClient
var mockGH *mockGitHubClient
var mockProj *mockProjectClient

type mergeHelper struct{}

type mergeOption func(*WorkflowMergeCmd)

var merge = &mergeHelper{}

func (h *mergeHelper) withMocks(opts ...mergeOption) *WorkflowMergeCmd {
	mockWksp = &mockWorkspaceSetupClient{}
	mockGit = &mockGitClient{}
	mockGH = &mockGitHubClient{}
	mockProj = &mockProjectClient{}
	cmd := &WorkflowMergeCmd{
		workspace: mockWksp,
		git:       mockGit,
		github:    mockGH,
		project:   mockProj,
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

func (h *mergeHelper) withWorkspace(wc WorkspaceSetupClient) mergeOption {
	return func(cmd *WorkflowMergeCmd) {
		cmd.workspace = wc
		if m, ok := wc.(*mockWorkspaceSetupClient); ok {
			mockWksp = m
		}
	}
}

func (h *mergeHelper) withGit(gc GitClient) mergeOption {
	return func(cmd *WorkflowMergeCmd) {
		cmd.git = gc
		if m, ok := gc.(*mockGitClient); ok {
			mockGit = m
		}
	}
}

func (h *mergeHelper) withGitHub(gc GitHubClient) mergeOption {
	return func(cmd *WorkflowMergeCmd) {
		cmd.github = gc
		if m, ok := gc.(*mockGitHubClient); ok {
			mockGH = m
		}
	}
}

func (h *mergeHelper) withProject(pc ProjectClient) mergeOption {
	return func(cmd *WorkflowMergeCmd) {
		cmd.project = pc
		if m, ok := pc.(*mockProjectClient); ok {
			mockProj = m
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

type gitHelper struct{}

var git = &gitHelper{}

func (h *gitHelper) commitAndPushCalled() bool {
	return mockGit != nil && mockGit.commitAndPushCalled
}

type githubHelper struct{}

var github = &githubHelper{}

func (h *githubHelper) waitForHeadSyncCalled() bool {
	return mockGH != nil && mockGH.waitForHeadSyncCalled
}

func (h *githubHelper) mergePRCalled() bool {
	return mockGH != nil && mockGH.mergePRCalled
}

func (h *githubHelper) thatTimesOutHeadSync() *mockGitHubClient {
	return &mockGitHubClient{
		waitForHeadSyncFunc: func(string) error { return errMock },
	}
}

type projectHelper struct{}

var project = &projectHelper{}

func (h *projectHelper) loadAllCalled() bool {
	return mockProj != nil && mockProj.loadAllCalled
}

func (h *projectHelper) deletedAll() bool {
	return mockProj != nil && mockProj.deleteAllCalled
}

func (h *projectHelper) withNoCompletedProjects() *mockProjectClient {
	return &mockProjectClient{
		loadAllFunc: func() ([]*ralphproj.Project, error) {
			return []*ralphproj.Project{
				{Slug: "req-1", Requirements: []ralphproj.Requirement{{Slug: "req-1", Passing: false}}},
			}, nil
		},
		filterPassingFunc: func(projects []*ralphproj.Project) []*ralphproj.Project {
			return nil
		},
	}
}

func (h *projectHelper) withCompletedProjects() *mockProjectClient {
	return &mockProjectClient{
		loadAllFunc: func() ([]*ralphproj.Project, error) {
			return []*ralphproj.Project{
				{Slug: "req-1", Requirements: []ralphproj.Requirement{{Slug: "req-1", Passing: false}}},
				{Slug: "req-2", Requirements: []ralphproj.Requirement{{Slug: "req-2", Passing: true}}},
			}, nil
		},
		filterPassingFunc: func(projects []*ralphproj.Project) []*ralphproj.Project {
			var passing []*ralphproj.Project
			for _, p := range projects {
				allPassing := true
				for _, req := range p.Requirements {
					if !req.Passing {
						allPassing = false
						break
					}
				}
				if allPassing {
					passing = append(passing, p)
				}
			}
			return passing
		},
	}
}

type flagsHelper struct{}

var flags = &flagsHelper{}

func (h *flagsHelper) any() WorkflowMergeFlags {
	return WorkflowMergeFlags{
		Repo:        "owner/repo",
		CloneBranch: "main",
		PRBranch:    "feature/test",
		PRNumber:    42,
		BotName:     "ralph",
		BotEmail:    "ralph@example.com",
	}
}
