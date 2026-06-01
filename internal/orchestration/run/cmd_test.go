package run

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
)

// ---------------------------------------------------------------------------
// Mock types for RunCmd clients
// ---------------------------------------------------------------------------

type mockWorkspaceClient struct {
	ChangeDirectoryFunc func(string) error
	ChangedDir          string
	ChangeDirCalled     bool
}

func (m *mockWorkspaceClient) ChangeDirectory(path string) error {
	m.ChangeDirCalled = true
	m.ChangedDir = path
	if m.ChangeDirectoryFunc != nil {
		return m.ChangeDirectoryFunc(path)
	}
	return nil
}

type mockConfigClient struct {
	LoadFunc   func() (*config.RalphConfig, error)
	LoadCalled bool
}

func (m *mockConfigClient) Load() (*config.RalphConfig, error) {
	m.LoadCalled = true
	if m.LoadFunc != nil {
		return m.LoadFunc()
	}
	return config.Any(), nil
}

type mockProjectRepo struct {
	LoadFunc          func(string) (*project.Project, error)
	ValidateFileFunc  func(string) error
	LoadCalled        bool
	ValidateFileCalled bool
	Project           *project.Project
	Err               error
}

func (m *mockProjectRepo) Load(path string) (*project.Project, error) {
	m.LoadCalled = true
	if m.LoadFunc != nil {
		return m.LoadFunc(path)
	}
	if m.Err != nil {
		return nil, m.Err
	}
	if m.Project != nil {
		return m.Project, nil
	}
	return project.Any(), nil
}

func (m *mockProjectRepo) ValidateFile(path string) error {
	m.ValidateFileCalled = true
	if m.ValidateFileFunc != nil {
		return m.ValidateFileFunc(path)
	}
	return nil
}

type mockLocalRunnerClient struct {
	RunLocalFunc    func(*project.Project, *config.RalphConfig) error
	LastProject     *project.Project
	LastConfig      *config.RalphConfig
	RunLocalCalled  bool
}

func (m *mockLocalRunnerClient) RunLocal(proj *project.Project, cfg *config.RalphConfig) error {
	m.RunLocalCalled = true
	m.LastProject = proj
	m.LastConfig = cfg
	if m.RunLocalFunc != nil {
		return m.RunLocalFunc(proj, cfg)
	}
	return nil
}

type mockRemoteRunnerClient struct {
	RunRemoteFunc    func(*project.Project, bool) error
	LastProject      *project.Project
	LastFollow       bool
	RunRemoteCalled  bool
}

func (m *mockRemoteRunnerClient) RunRemote(proj *project.Project, follow bool) error {
	m.RunRemoteCalled = true
	m.LastProject = proj
	m.LastFollow = follow
	if m.RunRemoteFunc != nil {
		return m.RunRemoteFunc(proj, follow)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Option types and helpers for building a RunCmd with mocks
// ---------------------------------------------------------------------------

type cmdOption func(*RunCmd)

func cmdWithWorkspace(w WorkspaceClient) cmdOption {
	return func(c *RunCmd) { c.workspace = w }
}

func cmdWithConfig(cfg ConfigClient) cmdOption {
	return func(c *RunCmd) { c.config = cfg }
}

func cmdWithProject(pc ProjectRepo) cmdOption {
	return func(c *RunCmd) { c.project = pc }
}

func cmdWithGit(gc GitClient) cmdOption {
	return func(c *RunCmd) { c.git = gc }
}

func cmdWithLocal(l LocalRunnerClient) cmdOption {
	return func(c *RunCmd) { c.local = l }
}

func cmdWithRemote(r RemoteRunnerClient) cmdOption {
	return func(c *RunCmd) { c.remote = r }
}

func cmdWithMocks(opts ...cmdOption) *RunCmd {
	cmd := &RunCmd{
		workspace: &mockWorkspaceClient{},
		config:    &mockConfigClient{},
		project:   &mockProjectRepo{},
		git:       &git.MockClient{},
		local:     &mockLocalRunnerClient{},
		remote:    &mockRemoteRunnerClient{},
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

// ---------------------------------------------------------------------------
// Flag helpers
// ---------------------------------------------------------------------------

func flagsAny() RunFlags {
	return RunFlags{ProjectFile: "/fake/project.yaml"}
}

func flagsWithNoBase() RunFlags {
	return RunFlags{ProjectFile: "/fake/project.yaml"}
}

func flagsWithMaxIterations(n int) RunFlags {
	return RunFlags{ProjectFile: "/fake/project.yaml", MaxIterations: n}
}

func flagsWithLocal() RunFlags {
	return RunFlags{ProjectFile: "/fake/project.yaml", Local: true}
}

func flagsWithFollowAndLocal() RunFlags {
	return RunFlags{ProjectFile: "/fake/project.yaml", Follow: true, Local: true}
}

func flagsWithDebugAndLocal() RunFlags {
	return RunFlags{ProjectFile: "/fake/project.yaml", Debug: "feature-x", Local: true}
}

func flagsWithWorkingDir(dir string) RunFlags {
	return RunFlags{ProjectFile: "/fake/project.yaml", WorkingDir: dir}
}

// ---------------------------------------------------------------------------
// Config mock builders
// ---------------------------------------------------------------------------

func workspaceThatFailsChangeDirectory() WorkspaceClient {
	return &mockWorkspaceClient{
		ChangeDirectoryFunc: func(string) error {
			return errors.New("workspace change failed")
		},
	}
}

func configThatFailsLoad() ConfigClient {
	return &mockConfigClient{
		LoadFunc: func() (*config.RalphConfig, error) {
			return nil, errors.New("config load failed")
		},
	}
}

func configWithMaxIterations(n int) ConfigClient {
	cfg := config.Any()
	cfg.MaxIterations = n
	return &mockConfigClient{
		LoadFunc: func() (*config.RalphConfig, error) { return cfg, nil },
	}
}

// ---------------------------------------------------------------------------
// Project mock builders
// ---------------------------------------------------------------------------

func projectThatFailsValidation() ProjectRepo {
	return &mockProjectRepo{
		ValidateFileFunc: func(string) error {
			return errors.New("project file not found")
		},
	}
}

func projectThatFailsLoad() ProjectRepo {
	return &mockProjectRepo{
		LoadFunc: func(string) (*project.Project, error) {
			return nil, errors.New("project load failed")
		},
	}
}

func projectWithSlug(slug string) ProjectRepo {
	return &mockProjectRepo{
		Project: &project.Project{Slug: slug},
	}
}

// ---------------------------------------------------------------------------
// Git mock builders
// ---------------------------------------------------------------------------

func gitOnBranch(branch string) GitClient {
	return &git.MockClient{
		CurrentBranchFunc: func() (string, error) { return branch, nil },
	}
}

// ---------------------------------------------------------------------------
// Accessor helpers for mock queries
// ---------------------------------------------------------------------------

func projectLoaded(cmd *RunCmd) bool {
	if m, ok := cmd.project.(*mockProjectRepo); ok {
		return m.LoadCalled
	}
	return false
}

func gitCurrentBranchCalled(cmd *RunCmd) bool {
	if m, ok := cmd.git.(*git.MockClient); ok {
		return m.CurrentBranchCalled
	}
	return false
}

func remoteLastProject(cmd *RunCmd) *project.Project {
	if m, ok := cmd.remote.(*mockRemoteRunnerClient); ok {
		return m.LastProject
	}
	return nil
}

func projectFileValidated(cmd *RunCmd) bool {
	if m, ok := cmd.project.(*mockProjectRepo); ok {
		return m.ValidateFileCalled
	}
	return false
}

func configLoaded(cmd *RunCmd) bool {
	if m, ok := cmd.config.(*mockConfigClient); ok {
		return m.LoadCalled
	}
	return false
}

func localRunLocalCalled(cmd *RunCmd) bool {
	if m, ok := cmd.local.(*mockLocalRunnerClient); ok {
		return m.RunLocalCalled
	}
	return false
}

func remoteRunRemoteCalled(cmd *RunCmd) bool {
	if m, ok := cmd.remote.(*mockRemoteRunnerClient); ok {
		return m.RunRemoteCalled
	}
	return false
}

// ---------------------------------------------------------------------------
// Tests: prepareSetup
// ---------------------------------------------------------------------------

func TestPrepareSetupConfigLoadFailureAbortsEarly(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithConfig(configThatFailsLoad()),
	)
	err := cmd.Run(flagsAny())
	require.Error(t, err)
	require.False(t, projectLoaded(cmd))
}

func TestPrepareSetupProjectLoadFailureAbortsEarly(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithProject(projectThatFailsLoad()),
	)
	err := cmd.Run(flagsAny())
	require.Error(t, err)
	require.False(t, gitCurrentBranchCalled(cmd))
}

func TestPrepareSetupBaseBranchFromCurrentWhenDifferentFromProject(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithGit(gitOnBranch("feature-x")),
		cmdWithProject(projectWithSlug("my-project")),
	)
	err := cmd.Run(flagsWithNoBase())
	require.NoError(t, err)
	require.Equal(t, "feature-x", remoteLastProject(cmd).BaseBranch)
}

func TestPrepareSetupMaxIterationsFlagOverridesConfig(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithConfig(configWithMaxIterations(5)),
	)
	err := cmd.Run(flagsWithMaxIterations(2))
	require.NoError(t, err)
	require.Equal(t, 2, remoteLastProject(cmd).MaxIterations)
}

// ---------------------------------------------------------------------------
// Scenario tests: Working directory changed before project file loaded
// ---------------------------------------------------------------------------

func TestRunWorkingDirectoryChangedBeforeProjectFileLoaded(t *testing.T) {
	ws := &mockWorkspaceClient{}
	proj := &mockProjectRepo{}
	cmd := cmdWithMocks(
		cmdWithWorkspace(ws),
		cmdWithProject(proj),
	)
	err := cmd.Run(flagsWithWorkingDir("/path/to/project"))
	require.NoError(t, err)
	require.True(t, ws.ChangeDirCalled)
	require.Equal(t, "/path/to/project", ws.ChangedDir)
	require.True(t, proj.ValidateFileCalled)
}

// ---------------------------------------------------------------------------
// Scenario tests: Project file not found error message
// ---------------------------------------------------------------------------

func TestRunProjectFileNotFoundErrorMessage(t *testing.T) {
	proj := &mockProjectRepo{
		ValidateFileFunc: func(string) error {
			return errors.New("project file not found: /nonexistent.yaml")
		},
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
	)
	err := cmd.Run(flagsAny())
	require.Error(t, err)
	require.Contains(t, err.Error(), "project file not found")
	require.False(t, configLoaded(cmd))
}

// ---------------------------------------------------------------------------
// Scenario tests: --follow with --local rejected
// ---------------------------------------------------------------------------

func TestRunFollowWithLocalRejected(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithFollowAndLocal())
	require.Error(t, err)
	require.Contains(t, err.Error(), "--follow flag is not applicable with --local flag")
}

// ---------------------------------------------------------------------------
// Scenario tests: --debug with --local rejected
// ---------------------------------------------------------------------------

func TestRunDebugWithLocalRejected(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithDebugAndLocal())
	require.Error(t, err)
	require.Contains(t, err.Error(), "--debug flag is not applicable with --local flag")
}

// ---------------------------------------------------------------------------
// Tests: Model and Context included in ExecutionSetup (item)
// ---------------------------------------------------------------------------

func TestPrepareSetupIncludesModelAndContext(t *testing.T) {
	cmd := cmdWithMocks()
	flags := RunFlags{
		ProjectFile: "/fake/project.yaml",
		Model:       "gpt-4",
		Context:     "my-cluster",
	}
	setup, err := cmd.prepareSetup(flags)
	require.NoError(t, err)
	require.Equal(t, "gpt-4", setup.Model)
	require.Equal(t, "my-cluster", setup.Context)
}

// ---------------------------------------------------------------------------
// Tests: git.SanitizeBranchName scenario tests
// ---------------------------------------------------------------------------

func TestBranchNameSlugWithSpacesAndCapitals(t *testing.T) {
	result := git.SanitizeBranchName("My Feature Work")
	require.Equal(t, "my-feature-work", result)
}

func TestBranchNameSlugWithSpecialCharacters(t *testing.T) {
	result := git.SanitizeBranchName("fix: auth/bug")
	require.Equal(t, "fix-authbug", result)
}

func TestBranchNameEmptySlug(t *testing.T) {
	result := git.SanitizeBranchName("")
	require.Equal(t, "unnamed-project", result)
}

func TestBranchNameAllInvalidCharacters(t *testing.T) {
	result := git.SanitizeBranchName("!!!@@@###")
	require.Equal(t, "unnamed-project", result)
}

// ---------------------------------------------------------------------------
// Tests: RunCmd dispatch
// ---------------------------------------------------------------------------

func TestRunLocalDispatchesToLocalRunner(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithLocal())
	require.NoError(t, err)
	require.True(t, localRunLocalCalled(cmd))
	require.False(t, remoteRunRemoteCalled(cmd))
}

func TestRunRemoteDispatchesToRemoteRunner(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsAny())
	require.NoError(t, err)
	require.True(t, remoteRunRemoteCalled(cmd))
	require.False(t, localRunLocalCalled(cmd))
}

func TestRunWorkingDirectoryFailureAbortsEarly(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithWorkspace(workspaceThatFailsChangeDirectory()),
	)
	err := cmd.Run(flagsAny())
	require.Error(t, err)
	require.False(t, projectFileValidated(cmd))
}

func TestRunProjectFileNotFoundAbortsEarly(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithProject(projectThatFailsValidation()),
	)
	err := cmd.Run(flagsAny())
	require.Error(t, err)
	require.False(t, configLoaded(cmd))
}

func TestRunIncompatibleFlagsAbortBeforeSetup(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithFollowAndLocal())
	require.Error(t, err)
	require.False(t, projectLoaded(cmd))
}
