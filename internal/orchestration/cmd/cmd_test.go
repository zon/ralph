package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/project"
)

// Mock types ----------------------------------------------------------------

type mockWorkspaceClient struct {
	changeDirectoryFunc   func(dir string) error
	changeDirectoryCalled bool
}

func (m *mockWorkspaceClient) ChangeDirectory(dir string) error {
	m.changeDirectoryCalled = true
	if m.changeDirectoryFunc != nil {
		return m.changeDirectoryFunc(dir)
	}
	return nil
}

type mockProjectClient struct {
	validateFileFunc   func(path string) error
	validateFileCalled bool
	loadFunc           func(path string) (*project.Project, error)
	loadCalled         bool
}

func (m *mockProjectClient) ValidateFile(path string) error {
	m.validateFileCalled = true
	if m.validateFileFunc != nil {
		return m.validateFileFunc(path)
	}
	return nil
}

func (m *mockProjectClient) Load(path string) (*project.Project, error) {
	m.loadCalled = true
	if m.loadFunc != nil {
		return m.loadFunc(path)
	}
	return &project.Project{Slug: "test-project"}, nil
}

type mockConfigClient struct {
	loadFunc   func() (*config.RalphConfig, error)
	loadCalled bool
}

func (m *mockConfigClient) Load() (*config.RalphConfig, error) {
	m.loadCalled = true
	if m.loadFunc != nil {
		return m.loadFunc()
	}
	return &config.RalphConfig{DefaultBranch: "main", MaxIterations: 10}, nil
}

type mockGitClient struct {
	currentBranchFunc     func() (string, error)
	currentBranchCalled   bool
	branchNameFunc        func(slug string) string
}

func (m *mockGitClient) CurrentBranch() (string, error) {
	m.currentBranchCalled = true
	if m.currentBranchFunc != nil {
		return m.currentBranchFunc()
	}
	return "main", nil
}

func (m *mockGitClient) BranchName(slug string) string {
	if m.branchNameFunc != nil {
		return m.branchNameFunc(slug)
	}
	return slug
}

type mockRunnerClient struct {
	executeFunc   func(setup ExecutionSetup) error
	executeCalled bool
	lastSetup     ExecutionSetup
}

func (m *mockRunnerClient) Execute(setup ExecutionSetup) error {
	m.executeCalled = true
	m.lastSetup = setup
	if m.executeFunc != nil {
		return m.executeFunc(setup)
	}
	return nil
}

// Shared mock state ---------------------------------------------------------

type mocks struct {
	workspace WorkspaceClient
	project   ProjectClient
	config    ConfigClient
	git       GitClient
	runner    RunnerClient
}

var currentMocks *mocks

// Helper types --------------------------------------------------------------

type runHelper struct{}

func (runHelper) withMocks(opts ...func(*mocks)) *RunCmd {
	m := &mocks{}
	for _, opt := range opts {
		opt(m)
	}
	if m.workspace == nil {
		m.workspace = &mockWorkspaceClient{}
	}
	if m.project == nil {
		m.project = &mockProjectClient{}
	}
	if m.config == nil {
		m.config = &mockConfigClient{}
	}
	if m.git == nil {
		m.git = &mockGitClient{}
	}
	if m.runner == nil {
		m.runner = &mockRunnerClient{}
	}
	currentMocks = m
	return &RunCmd{
		workspace: m.workspace,
		project:   m.project,
		config:    m.config,
		git:       m.git,
		runner:    m.runner,
	}
}

func (runHelper) withWorkspace(wc WorkspaceClient) func(*mocks) {
	return func(m *mocks) {
		m.workspace = wc
	}
}

func (runHelper) withProject(pc ProjectClient) func(*mocks) {
	return func(m *mocks) {
		m.project = pc
	}
}

func (runHelper) withConfig(cc ConfigClient) func(*mocks) {
	return func(m *mocks) {
		m.config = cc
	}
}

func (runHelper) withGit(gc GitClient) func(*mocks) {
	return func(m *mocks) {
		m.git = gc
	}
}

type workspaceHelper struct{}

func (workspaceHelper) thatFailsChangeDirectory() WorkspaceClient {
	return &mockWorkspaceClient{
		changeDirectoryFunc: func(dir string) error {
			return errors.New("change directory failed")
		},
	}
}

type projectHelper struct{}

func (projectHelper) thatFailsValidation() ProjectClient {
	return &mockProjectClient{
		validateFileFunc: func(path string) error {
			return errors.New("project file not found")
		},
	}
}

func (projectHelper) thatFailsLoad() ProjectClient {
	return &mockProjectClient{
		loadFunc: func(path string) (*project.Project, error) {
			return nil, errors.New("project load failed")
		},
	}
}

func (projectHelper) withSlug(slug string) ProjectClient {
	return &mockProjectClient{
		loadFunc: func(path string) (*project.Project, error) {
			return &project.Project{Slug: slug}, nil
		},
	}
}

func (projectHelper) fileValidated() bool {
	if mc, ok := currentMocks.project.(*mockProjectClient); ok {
		return mc.validateFileCalled
	}
	return false
}

func (projectHelper) loaded() bool {
	if mc, ok := currentMocks.project.(*mockProjectClient); ok {
		return mc.loadCalled
	}
	return false
}

type configHelper struct{}

func (configHelper) thatFailsLoad() ConfigClient {
	return &mockConfigClient{
		loadFunc: func() (*config.RalphConfig, error) {
			return nil, errors.New("config load failed")
		},
	}
}

func (configHelper) withMaxIterations(n int) ConfigClient {
	return &mockConfigClient{
		loadFunc: func() (*config.RalphConfig, error) {
			return &config.RalphConfig{MaxIterations: n, DefaultBranch: "main"}, nil
		},
	}
}

func (configHelper) loaded() bool {
	if mc, ok := currentMocks.config.(*mockConfigClient); ok {
		return mc.loadCalled
	}
	return false
}

type flagsHelper struct{}

func (flagsHelper) any() RunFlags {
	return RunFlags{ProjectFile: "test.yaml"}
}

func (flagsHelper) withFollowAndLocal() RunFlags {
	return RunFlags{
		ProjectFile: "test.yaml",
		Follow:      true,
		Local:       true,
	}
}

func (flagsHelper) withNoBase() RunFlags {
	return RunFlags{ProjectFile: "test.yaml", Base: ""}
}

func (flagsHelper) withMaxIterations(n int) RunFlags {
	return RunFlags{ProjectFile: "test.yaml", MaxIterations: n}
}

type runnerHelper struct{}

func (runnerHelper) executeCalled() bool {
	if mc, ok := currentMocks.runner.(*mockRunnerClient); ok {
		return mc.executeCalled
	}
	return false
}

func (runnerHelper) lastSetup() ExecutionSetup {
	if mc, ok := currentMocks.runner.(*mockRunnerClient); ok {
		return mc.lastSetup
	}
	return ExecutionSetup{}
}

type gitHelper struct{}

func (gitHelper) onBranch(name string) GitClient {
	return &mockGitClient{
		currentBranchFunc: func() (string, error) {
			return name, nil
		},
	}
}

func (gitHelper) currentBranchCalled() bool {
	if mc, ok := currentMocks.git.(*mockGitClient); ok {
		return mc.currentBranchCalled
	}
	return false
}

func (flagsHelper) withWorkingDir(dir string) RunFlags {
	return RunFlags{WorkingDir: dir, ProjectFile: "test.yaml"}
}

func (flagsHelper) withDebugAndLocal(branch string) RunFlags {
	return RunFlags{
		ProjectFile: "test.yaml",
		Debug:       branch,
		Local:       true,
	}
}

// Package-level variables for test helper access ---------------------------

var (
	run       = runHelper{}
	workspace = workspaceHelper{}
	projectH  = projectHelper{}
	configH   = configHelper{}
	flags     = flagsHelper{}
	runner    = runnerHelper{}
	git       = gitHelper{}
)

// Tests from the requirement -----------------------------------------------

func TestRunWorkingDirectoryFailureAbortsEarly(t *testing.T) {
	cmd := run.withMocks(
		run.withWorkspace(workspace.thatFailsChangeDirectory()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, projectH.fileValidated())
}

func TestRunProjectFileNotFoundAbortsEarly(t *testing.T) {
	cmd := run.withMocks(
		run.withProject(projectH.thatFailsValidation()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.False(t, configH.loaded())
}

func TestRunIncompatibleFlagsAbortBeforeSetup(t *testing.T) {
	cmd := run.withMocks()
	err := cmd.Run(flags.withFollowAndLocal())
	require.Error(t, err)
	require.False(t, configH.loaded())
}

func TestRunDispatchesWithPreparedSetup(t *testing.T) {
	cmd := run.withMocks()
	err := cmd.Run(flags.any())
	require.NoError(t, err)
	require.True(t, runner.executeCalled())
}

// Scenario tests ------------------------------------------------------------

func TestRunScenario_WorkingDirChangedBeforeProjectFileLoaded(t *testing.T) {
	wc := &mockWorkspaceClient{}
	cmd := run.withMocks(
		run.withWorkspace(wc),
	)
	err := cmd.Run(flags.withWorkingDir("/path/to/project"))
	require.NoError(t, err)
	require.True(t, wc.changeDirectoryCalled)
}

func TestRunScenario_ProjectFileNotFound(t *testing.T) {
	cmd := run.withMocks(
		run.withProject(projectH.thatFailsValidation()),
	)
	err := cmd.Run(flags.any())
	require.Error(t, err)
	require.Contains(t, err.Error(), "project file not found")
	require.False(t, runner.executeCalled())
}

func TestRunScenario_FollowWithLocal(t *testing.T) {
	cmd := run.withMocks()
	err := cmd.Run(flags.withFollowAndLocal())
	require.Error(t, err)
	require.Contains(t, err.Error(), "--follow flag is not applicable with --local flag")
	require.False(t, runner.executeCalled())
}

func TestRunScenario_DebugWithLocal(t *testing.T) {
	cmd := run.withMocks()
	err := cmd.Run(flags.withDebugAndLocal("my-branch"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "--debug flag is not applicable with --local flag")
	require.False(t, runner.executeCalled())
}
