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

type mockProjectRepo struct {
	ResolveInputFileFunc  func(string) (*project.InputFile, error)
	ResolveInputFileCalled bool
	InputFile             *project.InputFile
	Err                   error
}

func (m *mockProjectRepo) ResolveInputFile(path string) (*project.InputFile, error) {
	m.ResolveInputFileCalled = true
	if m.ResolveInputFileFunc != nil {
		return m.ResolveInputFileFunc(path)
	}
	if m.Err != nil {
		return nil, m.Err
	}
	if m.InputFile != nil {
		return m.InputFile, nil
	}
	return project.ForProjectInput(project.Any()), nil
}

type mockLocalRunnerClient struct {
	RunLocalFunc    func(*project.InputFile, *config.RalphConfig, string) error
	LastInput       *project.InputFile
	LastConfig      *config.RalphConfig
	LastBaseBranch  string
	RunLocalCalled  bool
}

func (m *mockLocalRunnerClient) RunLocal(input *project.InputFile, cfg *config.RalphConfig, baseBranch string) error {
	m.RunLocalCalled = true
	m.LastInput = input
	m.LastConfig = cfg
	m.LastBaseBranch = baseBranch
	if m.RunLocalFunc != nil {
		return m.RunLocalFunc(input, cfg, baseBranch)
	}
	return nil
}

type mockRemoteRunnerClient struct {
	RunFunc    func(*project.InputFile, RunRemoteFlags) error
	LastInput  *project.InputFile
	LastFlags  RunRemoteFlags
	RunCalled  bool
}

func (m *mockRemoteRunnerClient) Run(input *project.InputFile, flags RunRemoteFlags) error {
	m.RunCalled = true
	m.LastInput = input
	m.LastFlags = flags
	if m.RunFunc != nil {
		return m.RunFunc(input, flags)
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

func cmdWithConfig(cfg config.Loader) cmdOption {
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
		config:    &config.MockLoader{},
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
	return RunFlags{InputFile: "/fake/project.yaml"}
}

func flagsWithNoBase() RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml"}
}

func flagsWithMaxIterations(n int) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", MaxIterations: n}
}

func flagsWithExtraIterations(n int) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", ExtraIterations: n}
}

func flagsWithLocal() RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", Local: true}
}

func flagsWithFollowAndLocal() RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", Follow: true, Local: true}
}

func flagsWithDebugAndLocal() RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", Debug: "feature-x", Local: true}
}

func flagsWithWorkingDir(dir string) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", WorkingDir: dir}
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

func configThatFailsLoad() config.Loader {
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) {
			return nil, errors.New("config load failed")
		},
	}
}

func configWithMaxIterations(n int) config.Loader {
	cfg := config.Any()
	cfg.MaxIterations = n
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) { return cfg, nil },
	}
}

// ---------------------------------------------------------------------------
// Project mock builders
// ---------------------------------------------------------------------------

func projectThatFailsResolve() ProjectRepo {
	return &mockProjectRepo{
		ResolveInputFileFunc: func(string) (*project.InputFile, error) {
			return nil, errors.New("input file not found: /nonexistent.yaml")
		},
	}
}

func projectThatFailsLoad() ProjectRepo {
	return &mockProjectRepo{
		ResolveInputFileFunc: func(string) (*project.InputFile, error) {
			return nil, errors.New("project load failed")
		},
	}
}

func projectWithSlug(slug string) ProjectRepo {
	return &mockProjectRepo{
		InputFile: project.ForProjectInput(&project.Project{Slug: slug}),
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

func inputResolved(cmd *RunCmd) bool {
	if m, ok := cmd.project.(*mockProjectRepo); ok {
		return m.ResolveInputFileCalled
	}
	return false
}

func remoteLastInput(cmd *RunCmd) *project.InputFile {
	if m, ok := cmd.remote.(*mockRemoteRunnerClient); ok {
		return m.LastInput
	}
	return nil
}

func localRunLocalCalled(cmd *RunCmd) bool {
	if m, ok := cmd.local.(*mockLocalRunnerClient); ok {
		return m.RunLocalCalled
	}
	return false
}

func remoteRunCalled(cmd *RunCmd) bool {
	if m, ok := cmd.remote.(*mockRemoteRunnerClient); ok {
		return m.RunCalled
	}
	return false
}

func localLastInput(cmd *RunCmd) *project.InputFile {
	if m, ok := cmd.local.(*mockLocalRunnerClient); ok {
		return m.LastInput
	}
	return nil
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
}

func TestPrepareSetupProjectLoadFailureAbortsEarly(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithProject(projectThatFailsLoad()),
	)
	err := cmd.Run(flagsAny())
	require.Error(t, err)
}

func TestPrepareSetupBaseBranchFromCurrentWhenDifferentFromProject(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithGit(gitOnBranch("feature-x")),
		cmdWithProject(projectWithSlug("my-project")),
	)
	setup, err := cmd.prepareSetup(flagsWithNoBase(), project.ForProjectInput(&project.Project{Slug: "my-project"}))
	require.NoError(t, err)
	require.Equal(t, "feature-x", setup.BaseBranch)
}

func TestPrepareSetupMaxIterationsFlagOverridesConfig(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithConfig(configWithMaxIterations(5)),
	)
	setup, err := cmd.prepareSetup(flagsWithMaxIterations(2), project.ForProjectInput(project.Any()))
	require.NoError(t, err)
	require.Equal(t, 2, setup.MaxIterations)
}

func configWithExtraIterations(n int) config.Loader {
	cfg := config.Any()
	v := n
	cfg.ExtraIterations = &v
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) { return cfg, nil },
	}
}

func TestPrepareSetupExtraIterationsFlagOverridesConfig(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithConfig(configWithExtraIterations(5)),
	)
	flags := flagsWithExtraIterations(2)
	setup, err := cmd.prepareSetup(flags, project.ForProjectInput(project.Any()))
	require.NoError(t, err)
	require.NotNil(t, setup.Config.ExtraIterations)
	require.Equal(t, 2, *setup.Config.ExtraIterations)
}

func TestPrepareSetupExtraIterationsZeroDoesNotOverrideConfig(t *testing.T) {
	v := 5
	cfg := config.Any()
	cfg.ExtraIterations = &v
	cmd := cmdWithMocks(
		cmdWithConfig(&config.MockLoader{
			LoadFn: func() (*config.RalphConfig, error) { return cfg, nil },
		}),
	)
	flags := flagsWithExtraIterations(0)
	setup, err := cmd.prepareSetup(flags, project.ForProjectInput(project.Any()))
	require.NoError(t, err)
	require.NotNil(t, setup.Config.ExtraIterations)
	require.Equal(t, 5, *setup.Config.ExtraIterations)
}

func TestPrepareSetupExtraIterationsDefaultsToConfigWhenFlagAbsent(t *testing.T) {
	v := 3
	cfg := config.Any()
	cfg.ExtraIterations = &v
	cmd := cmdWithMocks(
		cmdWithConfig(&config.MockLoader{
			LoadFn: func() (*config.RalphConfig, error) { return cfg, nil },
		}),
	)
	flags := flagsAny()
	setup, err := cmd.prepareSetup(flags, project.ForProjectInput(project.Any()))
	require.NoError(t, err)
	require.NotNil(t, setup.Config.ExtraIterations)
	require.Equal(t, 3, *setup.Config.ExtraIterations)
}

// ---------------------------------------------------------------------------
// Scenario tests: Working directory changed before input file resolved
// ---------------------------------------------------------------------------

func TestRunWorkingDirectoryChangedBeforeInputFileResolved(t *testing.T) {
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
	require.True(t, proj.ResolveInputFileCalled)
}

// ---------------------------------------------------------------------------
// Scenario tests: Input file not found error message
// ---------------------------------------------------------------------------

func TestRunInputFileNotFoundErrorMessage(t *testing.T) {
	proj := &mockProjectRepo{
		ResolveInputFileFunc: func(string) (*project.InputFile, error) {
			return nil, errors.New("input file not found: /nonexistent.yaml")
		},
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
	)
	err := cmd.Run(flagsAny())
	require.Error(t, err)
	require.Contains(t, err.Error(), "input file not found")
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
		InputFile: "/fake/project.yaml",
		Model:     "gpt-4",
		Context:   "my-cluster",
	}
	setup, err := cmd.prepareSetup(flags, project.ForProjectInput(project.Any()))
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
	require.False(t, remoteRunCalled(cmd))
}

func TestRunRemoteDispatchesToRemoteRunner(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsAny())
	require.NoError(t, err)
	require.True(t, remoteRunCalled(cmd))
	require.False(t, localRunLocalCalled(cmd))
}

func TestRunWorkingDirectoryFailureAbortsEarly(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithWorkspace(workspaceThatFailsChangeDirectory()),
	)
	err := cmd.Run(flagsAny())
	require.Error(t, err)
	require.False(t, inputResolved(cmd))
}

func TestRunInputFileNotFoundAbortsEarly(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithProject(projectThatFailsResolve()),
	)
	err := cmd.Run(flagsAny())
	require.Error(t, err)
}

func TestRunIncompatibleFlagsAbortBeforeSetup(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithFollowAndLocal())
	require.Error(t, err)
	require.False(t, localRunLocalCalled(cmd))
	require.False(t, remoteRunCalled(cmd))
}

// ---------------------------------------------------------------------------
// Tests: Orchestration and spec inputs dispatch through RunCmd
// ---------------------------------------------------------------------------

func TestRunLocalDispatchesWithOrchestrationInput(t *testing.T) {
	proj := &mockProjectRepo{
		InputFile: project.ForOrchestrationInput("specs/features/ralph/run/orchestration.md"),
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
		cmdWithLocal(&mockLocalRunnerClient{}),
	)
	err := cmd.Run(flagsWithLocal())
	require.NoError(t, err)
	require.True(t, localRunLocalCalled(cmd))
	require.NotNil(t, localLastInput(cmd))
	require.True(t, localLastInput(cmd).IsOrchestration())
}

func TestRunLocalDispatchesWithSpecInput(t *testing.T) {
	proj := &mockProjectRepo{
		InputFile: project.ForSpecInput("specs/features/ralph/run/spec.md"),
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
		cmdWithLocal(&mockLocalRunnerClient{}),
	)
	err := cmd.Run(flagsWithLocal())
	require.NoError(t, err)
	require.True(t, localRunLocalCalled(cmd))
	require.NotNil(t, localLastInput(cmd))
	require.True(t, localLastInput(cmd).IsSpec())
}

func TestRunRemoteDispatchesWithOrchestrationInput(t *testing.T) {
	proj := &mockProjectRepo{
		InputFile: project.ForOrchestrationInput("specs/features/ralph/run/orchestration.md"),
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
		cmdWithRemote(&mockRemoteRunnerClient{}),
	)
	err := cmd.Run(flagsAny())
	require.NoError(t, err)
	require.True(t, remoteRunCalled(cmd))
	require.NotNil(t, remoteLastInput(cmd))
	require.True(t, remoteLastInput(cmd).IsOrchestration())
}

func TestRunRemoteDispatchesWithSpecInput(t *testing.T) {
	proj := &mockProjectRepo{
		InputFile: project.ForSpecInput("specs/features/ralph/run/spec.md"),
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
		cmdWithRemote(&mockRemoteRunnerClient{}),
	)
	err := cmd.Run(flagsAny())
	require.NoError(t, err)
	require.True(t, remoteRunCalled(cmd))
	require.NotNil(t, remoteLastInput(cmd))
	require.True(t, remoteLastInput(cmd).IsSpec())
}

// ---------------------------------------------------------------------------
// Tests: Input file not found aborts before flag validation and setup
// ---------------------------------------------------------------------------

func TestRunInputFileNotFoundAbortsBeforeFlagValidation(t *testing.T) {
	proj := &mockProjectRepo{
		ResolveInputFileFunc: func(string) (*project.InputFile, error) {
			return nil, errors.New("input file not found: /nonexistent.yaml")
		},
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
	)
	err := cmd.Run(flagsWithFollowAndLocal())
	require.Error(t, err)
	require.Contains(t, err.Error(), "input file not found")
}

// ---------------------------------------------------------------------------
// Tests: Incompatible flags rejected before setup
// ---------------------------------------------------------------------------

func TestRunIncompatibleFlagsRejectedBeforeSetupForProjectInput(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithFollowAndLocal())
	require.Error(t, err)
	require.False(t, localRunLocalCalled(cmd))
	require.False(t, remoteRunCalled(cmd))
}

// ---------------------------------------------------------------------------
// Tests: prepareSetup with non-project inputs
// ---------------------------------------------------------------------------

func TestPrepareSetupWithOrchestrationInputResolvesBaseBranch(t *testing.T) {
	cmd := cmdWithMocks()
	input := project.ForOrchestrationInput("specs/features/ralph/run/orchestration.md")
	setup, err := cmd.prepareSetup(flagsAny(), input)
	require.NoError(t, err)
	require.Equal(t, "main", setup.BaseBranch)
}

func TestPrepareSetupWithSpecInputResolvesBaseBranch(t *testing.T) {
	cmd := cmdWithMocks()
	input := project.ForSpecInput("specs/features/ralph/run/spec.md")
	setup, err := cmd.prepareSetup(flagsAny(), input)
	require.NoError(t, err)
	require.Equal(t, "main", setup.BaseBranch)
}
