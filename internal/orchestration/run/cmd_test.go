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
	ChangedDirs         []string
}

func (m *mockWorkspaceClient) ChangeDirectory(path string) error {
	m.ChangeDirCalled = true
	m.ChangedDir = path
	m.ChangedDirs = append(m.ChangedDirs, path)
	if m.ChangeDirectoryFunc != nil {
		return m.ChangeDirectoryFunc(path)
	}
	return nil
}

type mockProjectRepo struct {
	ResolveInputFileFunc   func(string) (*project.InputFile, error)
	ResolveInputFileCalled bool
	InputFile              *project.InputFile
	Err                    error
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
	RunLocalFunc             func(*project.InputFile, *config.RalphConfig) error
	RunLocalInWorktreeFunc   func(*project.InputFile, *config.RalphConfig) error
	LastInput                *project.InputFile
	LastConfig               *config.RalphConfig
	RunLocalCalled           bool
	RunLocalInWorktreeCalled bool
}

func (m *mockLocalRunnerClient) RunLocal(input *project.InputFile, cfg *config.RalphConfig) error {
	m.RunLocalCalled = true
	m.LastInput = input
	m.LastConfig = cfg
	if m.RunLocalFunc != nil {
		return m.RunLocalFunc(input, cfg)
	}
	return nil
}

func (m *mockLocalRunnerClient) RunLocalInWorktree(input *project.InputFile, cfg *config.RalphConfig) error {
	m.RunLocalInWorktreeCalled = true
	m.LastInput = input
	m.LastConfig = cfg
	if m.RunLocalInWorktreeFunc != nil {
		return m.RunLocalInWorktreeFunc(input, cfg)
	}
	return nil
}

type mockRemoteRunnerClient struct {
	RunFunc   func(*project.InputFile, RunRemoteFlags) error
	LastInput *project.InputFile
	LastFlags RunRemoteFlags
	RunCalled bool
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

type mockWorktreeClient struct {
	CreateWorktreeFunc             func(string, bool) (*git.WorktreeCommand, error)
	BranchCheckedOutInWorktreeFunc func(string, bool) (*git.WorktreeCommand, bool, error)
	RemoveWorktreeFunc             func(string, bool) (*git.WorktreeCommand, error)

	CreateWorktreeCalled             bool
	BranchCheckedOutInWorktreeCalled bool
	RemoveWorktreeCalled             bool
	CreateWorktreeBranch             string
	DetectedBranch                   string
	RemoveWorktreeBranch             string
	checkedOut                       bool
}

func (m *mockWorktreeClient) CreateWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error) {
	m.CreateWorktreeCalled = true
	m.CreateWorktreeBranch = branch
	if m.CreateWorktreeFunc != nil {
		return m.CreateWorktreeFunc(branch, dryRun)
	}
	return &git.WorktreeCommand{Args: []string{"worktree", "add", "-b", branch, "/sibling/repo-" + branch}, Path: "/sibling/repo-" + branch}, nil
}

func (m *mockWorktreeClient) BranchCheckedOutInWorktree(branch string, dryRun bool) (*git.WorktreeCommand, bool, error) {
	m.BranchCheckedOutInWorktreeCalled = true
	m.DetectedBranch = branch
	if m.BranchCheckedOutInWorktreeFunc != nil {
		return m.BranchCheckedOutInWorktreeFunc(branch, dryRun)
	}
	return &git.WorktreeCommand{Args: []string{"worktree", "list", "--porcelain"}}, m.checkedOut, nil
}

func (m *mockWorktreeClient) RemoveWorktree(branch string, dryRun bool) (*git.WorktreeCommand, error) {
	m.RemoveWorktreeCalled = true
	m.RemoveWorktreeBranch = branch
	if m.RemoveWorktreeFunc != nil {
		return m.RemoveWorktreeFunc(branch, dryRun)
	}
	return &git.WorktreeCommand{Args: []string{"worktree", "remove", "--force"}, Path: "/sibling/repo-" + branch}, nil
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

func cmdWithWorktree(wt WorktreeClient) cmdOption {
	return func(c *RunCmd) { c.worktree = wt }
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
		worktree:  &mockWorktreeClient{},
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

func flagsWithMode(mode string) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", Mode: mode}
}

func flagsWithFollow() RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", Follow: true}
}

func flagsWithFollowAndMode(mode string) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", Follow: true, Mode: mode}
}

func flagsWithDebugAndMode(mode string) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", Debug: "feature-x", Mode: mode}
}

func flagsWithExtraIterations(n int) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", ExtraIterations: n}
}

func flagsWithWorkingDir(dir string) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", WorkingDir: dir}
}

func flagsWithItems(query string) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", Items: query}
}

func flagsWithModeAndItems(mode, query string) RunFlags {
	return RunFlags{InputFile: "/fake/project.yaml", Mode: mode, Items: query}
}

func flagsWithCleanup() RunFlags {
	v := true
	return RunFlags{InputFile: "/fake/project.yaml", Cleanup: &v}
}

func flagsWithModeAndCleanup(mode string) RunFlags {
	v := true
	return RunFlags{InputFile: "/fake/project.yaml", Mode: mode, Cleanup: &v}
}

func flagsWithCleanupDisabled() RunFlags {
	v := false
	return RunFlags{InputFile: "/fake/project.yaml", Cleanup: &v}
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

func localRunLocalInWorktreeCalled(cmd *RunCmd) bool {
	if m, ok := cmd.local.(*mockLocalRunnerClient); ok {
		return m.RunLocalInWorktreeCalled
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

func configWithExtraIterations(n int) config.Loader {
	cfg := config.Any()
	v := n
	cfg.ExtraIterations = &v
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) { return cfg, nil },
	}
}

func configWithItems(query string) config.Loader {
	cfg := config.WithItems(query)
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) { return cfg, nil },
	}
}

func configWithCleanup() config.Loader {
	cfg := config.WithCleanup()
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) { return cfg, nil },
	}
}

func configWithMode(mode string) config.Loader {
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) { return config.WithMode(mode), nil },
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
	require.Equal(t, "/path/to/project", ws.ChangedDirs[0])
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
// Scenario tests: --follow rejected for local and worktree modes
// ---------------------------------------------------------------------------

func TestRunFollowRejectedForLocalAndWorktreeModes(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "local", mode: config.ModeLocal},
		{name: "worktree", mode: config.ModeWorktree},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdWithMocks()
			err := cmd.Run(flagsWithFollowAndMode(tt.mode))
			require.Error(t, err)
			require.Contains(t, err.Error(), "--follow flag is not applicable with --mode "+tt.mode)
			require.False(t, localRunLocalCalled(cmd))
			require.False(t, remoteRunCalled(cmd))
		})
	}
}

func TestRunFollowRejectedWhenWorktreeFromConfig(t *testing.T) {
	cmd := cmdWithMocks(cmdWithConfig(configWithMode(config.ModeWorktree)))
	err := cmd.Run(flagsWithFollow())
	require.Error(t, err)
	require.Contains(t, err.Error(), "--follow flag is not applicable with --mode worktree")
	require.False(t, localRunLocalCalled(cmd))
	require.False(t, remoteRunCalled(cmd))
}

// ---------------------------------------------------------------------------
// Scenario tests: --debug rejected for local and worktree modes
// ---------------------------------------------------------------------------

func TestRunDebugRejectedForLocalAndWorktreeModes(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "local", mode: config.ModeLocal},
		{name: "worktree", mode: config.ModeWorktree},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdWithMocks()
			err := cmd.Run(flagsWithDebugAndMode(tt.mode))
			require.Error(t, err)
			require.Contains(t, err.Error(), "--debug flag is not applicable with --mode "+tt.mode)
			require.False(t, localRunLocalCalled(cmd))
			require.False(t, remoteRunCalled(cmd))
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario tests: --follow and --debug accepted for remote mode
// ---------------------------------------------------------------------------

func TestRunFollowAndDebugAcceptedForRemoteMode(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithFollowAndMode(config.ModeRemote))
	require.NoError(t, err)
	require.True(t, remoteRunCalled(cmd))
	require.False(t, localRunLocalCalled(cmd))

	cmd = cmdWithMocks()
	err = cmd.Run(flagsWithDebugAndMode(config.ModeRemote))
	require.NoError(t, err)
	require.True(t, remoteRunCalled(cmd))
	require.False(t, localRunLocalCalled(cmd))
}

// ---------------------------------------------------------------------------
// Scenario tests: mode resolution and rejection
// ---------------------------------------------------------------------------

func TestRunModeResolvesFlagThenConfigThenLocal(t *testing.T) {
	t.Run("flag overrides configured mode", func(t *testing.T) {
		cmd := cmdWithMocks(cmdWithConfig(configWithMode(config.ModeRemote)))
		err := cmd.Run(flagsWithMode(config.ModeLocal))
		require.NoError(t, err)
		require.True(t, localRunLocalCalled(cmd))
		require.False(t, remoteRunCalled(cmd))
	})

	t.Run("configured mode used when no flag", func(t *testing.T) {
		cmd := cmdWithMocks(cmdWithConfig(configWithMode(config.ModeRemote)))
		err := cmd.Run(flagsAny())
		require.NoError(t, err)
		require.True(t, remoteRunCalled(cmd))
		require.False(t, localRunLocalCalled(cmd))
	})

	t.Run("local default when flag and config unset", func(t *testing.T) {
		cmd := cmdWithMocks()
		err := cmd.Run(flagsAny())
		require.NoError(t, err)
		require.True(t, localRunLocalCalled(cmd))
		require.False(t, remoteRunCalled(cmd))
		require.False(t, localRunLocalInWorktreeCalled(cmd))
	})
}

func TestRunInvalidModeRejectedBeforeExecution(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithMode("sandbox"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid mode: sandbox (expected local, worktree, or remote)")
	require.False(t, localRunLocalCalled(cmd))
	require.False(t, remoteRunCalled(cmd))
}

func TestRunInvalidConfiguredModeRejectedBeforeExecution(t *testing.T) {
	cmd := cmdWithMocks(cmdWithConfig(configWithMode("sandbox")))
	err := cmd.Run(flagsAny())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid mode: sandbox (expected local, worktree, or remote)")
	require.False(t, localRunLocalCalled(cmd))
	require.False(t, remoteRunCalled(cmd))
}

// ---------------------------------------------------------------------------
// Tests: Model and Context included in ExecutionSetup (item)
// ---------------------------------------------------------------------------

func TestPrepareSetupIncludesModelAndContext(t *testing.T) {
	cmd := cmdWithMocks()
	flags := RunFlags{
		InputFile: "/fake/project.yaml",
		Model:     "gpt-4",
		Agent:     "code-reviewer",
		Context:   "my-cluster",
	}
	setup, err := cmd.prepareSetup(flags, project.ForProjectInput(project.Any()))
	require.NoError(t, err)
	require.Equal(t, "gpt-4", setup.Model)
	require.Equal(t, "code-reviewer", setup.Agent)
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
	err := cmd.Run(flagsWithMode(config.ModeLocal))
	require.NoError(t, err)
	require.True(t, localRunLocalCalled(cmd))
	require.False(t, remoteRunCalled(cmd))
}

func TestRunRemoteDispatchesToRemoteRunner(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithMode(config.ModeRemote))
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
	err := cmd.Run(flagsWithFollowAndMode(config.ModeLocal))
	require.Error(t, err)
	require.False(t, localRunLocalCalled(cmd))
	require.False(t, remoteRunCalled(cmd))
}

// ---------------------------------------------------------------------------
// Tests: Orchestration and spec inputs dispatch through RunCmd
// ---------------------------------------------------------------------------

func TestRunLocalDispatchesWithOrchestrationInput(t *testing.T) {
	proj := &mockProjectRepo{
		InputFile: project.ForOrchestrationInput("specs/ralph/orchestration.md"),
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
		cmdWithLocal(&mockLocalRunnerClient{}),
	)
	err := cmd.Run(flagsWithMode(config.ModeLocal))
	require.NoError(t, err)
	require.True(t, localRunLocalCalled(cmd))
	require.NotNil(t, localLastInput(cmd))
	require.True(t, localLastInput(cmd).IsOrchestration())
}

func TestRunLocalDispatchesWithSpecInput(t *testing.T) {
	proj := &mockProjectRepo{
		InputFile: project.ForSpecInput("specs/ralph/run.md"),
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
		cmdWithLocal(&mockLocalRunnerClient{}),
	)
	err := cmd.Run(flagsWithMode(config.ModeLocal))
	require.NoError(t, err)
	require.True(t, localRunLocalCalled(cmd))
	require.NotNil(t, localLastInput(cmd))
	require.True(t, localLastInput(cmd).IsSpec())
}

func TestRunRemoteDispatchesWithOrchestrationInput(t *testing.T) {
	proj := &mockProjectRepo{
		InputFile: project.ForOrchestrationInput("specs/ralph/orchestration.md"),
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
		cmdWithRemote(&mockRemoteRunnerClient{}),
	)
	err := cmd.Run(flagsWithMode(config.ModeRemote))
	require.NoError(t, err)
	require.True(t, remoteRunCalled(cmd))
	require.NotNil(t, remoteLastInput(cmd))
	require.True(t, remoteLastInput(cmd).IsOrchestration())
}

func TestRunRemoteDispatchesWithSpecInput(t *testing.T) {
	proj := &mockProjectRepo{
		InputFile: project.ForSpecInput("specs/ralph/run.md"),
	}
	cmd := cmdWithMocks(
		cmdWithProject(proj),
		cmdWithRemote(&mockRemoteRunnerClient{}),
	)
	err := cmd.Run(flagsWithMode(config.ModeRemote))
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
	err := cmd.Run(flagsWithFollowAndMode(config.ModeLocal))
	require.Error(t, err)
	require.Contains(t, err.Error(), "input file not found")
}

// ---------------------------------------------------------------------------
// Tests: Incompatible flags rejected before setup
// ---------------------------------------------------------------------------

func TestRunIncompatibleFlagsRejectedBeforeSetupForProjectInput(t *testing.T) {
	cmd := cmdWithMocks()
	err := cmd.Run(flagsWithFollowAndMode(config.ModeLocal))
	require.Error(t, err)
	require.False(t, localRunLocalCalled(cmd))
	require.False(t, remoteRunCalled(cmd))
}

// ---------------------------------------------------------------------------
// Tests: prepareSetup with non-project inputs
// ---------------------------------------------------------------------------

func TestPrepareSetupWithOrchestrationInputResolvesBaseBranch(t *testing.T) {
	cmd := cmdWithMocks()
	input := project.ForOrchestrationInput("specs/ralph/orchestration.md")
	setup, err := cmd.prepareSetup(flagsAny(), input)
	require.NoError(t, err)
	require.Equal(t, "main", setup.BaseBranch)
}

func TestPrepareSetupWithSpecInputResolvesBaseBranch(t *testing.T) {
	cmd := cmdWithMocks()
	input := project.ForSpecInput("specs/ralph/run.md")
	setup, err := cmd.prepareSetup(flagsAny(), input)
	require.NoError(t, err)
	require.Equal(t, "main", setup.BaseBranch)
}

// ---------------------------------------------------------------------------
// Scenario tests: --items overrides the configured query
// ---------------------------------------------------------------------------

func TestRunItemsFlagOverridesConfiguredQuery(t *testing.T) {
	// GIVEN `items: .requirements` is set in `.ralph/config.yaml`
	cmd := cmdWithMocks(
		cmdWithConfig(configWithItems(".requirements")),
	)
	// AND the user passes `--items '.spec.tasks'`
	// WHEN the item query is resolved
	setup, err := cmd.prepareSetup(flagsWithItems(".spec.tasks"), project.ForProjectInput(project.Any()))
	require.NoError(t, err)
	// THEN the resolved query is `.spec.tasks`
	require.Equal(t, ".spec.tasks", setup.Config.Items)
}

// ---------------------------------------------------------------------------
// Scenario tests: resolved query passed to the execution mode
// ---------------------------------------------------------------------------

func TestRunResolvedQueryPassedToLocalRunner(t *testing.T) {
	// GIVEN the item query has been resolved locally
	local := &mockLocalRunnerClient{}
	cmd := cmdWithMocks(
		cmdWithConfig(configWithItems(".requirements")),
		cmdWithLocal(local),
	)
	// WHEN execution is dispatched to run-local
	err := cmd.Run(flagsWithModeAndItems(config.ModeLocal, ".spec.tasks"))
	require.NoError(t, err)
	// THEN the resolved query is passed down as a parameter
	require.True(t, local.RunLocalCalled)
	require.Equal(t, ".spec.tasks", local.LastConfig.Items)
}

func TestRunResolvedQueryPassedToRemoteRunner(t *testing.T) {
	// GIVEN the item query has been resolved locally
	remote := &mockRemoteRunnerClient{}
	cmd := cmdWithMocks(
		cmdWithConfig(configWithItems(".requirements")),
		cmdWithRemote(remote),
	)
	// WHEN execution is dispatched to run-remote
	err := cmd.Run(flagsWithModeAndItems(config.ModeRemote, ".spec.tasks"))
	require.NoError(t, err)
	// THEN the resolved query is passed down as a parameter
	require.True(t, remote.RunCalled)
	require.Equal(t, ".spec.tasks", remote.LastFlags.Items)
}

func TestRunLocalRunnerDoesNotReResolveQueryFromConfig(t *testing.T) {
	local := &mockLocalRunnerClient{}
	cmd := cmdWithMocks(
		cmdWithConfig(configWithItems(".requirements")),
		cmdWithLocal(local),
	)
	err := cmd.Run(flagsWithModeAndItems(config.ModeLocal, ".spec.tasks"))
	require.NoError(t, err)
	require.Equal(t, ".spec.tasks", local.LastConfig.Items)
}

func TestRunResolvedDefaultQueryPassedToRemoteRunner(t *testing.T) {
	remote := &mockRemoteRunnerClient{}
	cmd := cmdWithMocks(
		cmdWithConfig(configWithItems("")),
		cmdWithRemote(remote),
	)
	err := cmd.Run(flagsWithMode(config.ModeRemote))
	require.NoError(t, err)
	require.True(t, remote.RunCalled)
	require.Equal(t, ".", remote.LastFlags.Items)
}

// ---------------------------------------------------------------------------
// Scenario tests: --cleanup enables cleanup for one run
// ---------------------------------------------------------------------------

func TestRunCleanupFlagEnablesCleanup(t *testing.T) {
	// GIVEN `cleanup` is not set in `.ralph/config.yaml`
	cmd := cmdWithMocks()
	// AND the user passes `--cleanup`
	// WHEN cleanup is resolved
	setup, err := cmd.prepareSetup(flagsWithCleanup(), project.ForProjectInput(project.Any()))
	require.NoError(t, err)
	// THEN cleanup is enabled for this run
	require.True(t, setup.Config.Cleanup)
}

func TestRunCleanupFlagEnablesCleanupForLocalRun(t *testing.T) {
	local := &mockLocalRunnerClient{}
	cmd := cmdWithMocks(cmdWithLocal(local))
	err := cmd.Run(flagsWithModeAndCleanup(config.ModeLocal))
	require.NoError(t, err)
	require.True(t, local.LastConfig.Cleanup)
}

// ---------------------------------------------------------------------------
// Scenario tests: cleanup disabled by default
// ---------------------------------------------------------------------------

func TestRunCleanupDisabledByDefault(t *testing.T) {
	// GIVEN `cleanup` is not set in `.ralph/config.yaml`
	cmd := cmdWithMocks()
	// AND no `--cleanup` flag is passed
	// WHEN cleanup is resolved
	setup, err := cmd.prepareSetup(flagsAny(), project.ForProjectInput(project.Any()))
	require.NoError(t, err)
	// THEN cleanup is disabled and the project file survives the run
	require.False(t, setup.Config.Cleanup)
}

func TestRunCleanupDisabledByDefaultForLocalRun(t *testing.T) {
	local := &mockLocalRunnerClient{}
	cmd := cmdWithMocks(cmdWithLocal(local))
	err := cmd.Run(flagsWithMode(config.ModeLocal))
	require.NoError(t, err)
	require.False(t, local.LastConfig.Cleanup)
}

// ---------------------------------------------------------------------------
// Item tests: resolution order
// ---------------------------------------------------------------------------

func TestRunItemsResolvesFlagThenConfigThenDefault(t *testing.T) {
	t.Run("flag overrides configured query", func(t *testing.T) {
		cmd := cmdWithMocks(cmdWithConfig(configWithItems(".requirements")))
		setup, err := cmd.prepareSetup(flagsWithItems(".spec.tasks"), project.ForProjectInput(project.Any()))
		require.NoError(t, err)
		require.Equal(t, ".spec.tasks", setup.Config.Items)
	})

	t.Run("configured query used when no flag", func(t *testing.T) {
		cmd := cmdWithMocks(cmdWithConfig(configWithItems(".requirements")))
		setup, err := cmd.prepareSetup(flagsAny(), project.ForProjectInput(project.Any()))
		require.NoError(t, err)
		require.Equal(t, ".requirements", setup.Config.Items)
	})

	t.Run("default query used when flag and config unset", func(t *testing.T) {
		cmd := cmdWithMocks()
		setup, err := cmd.prepareSetup(flagsAny(), project.ForProjectInput(project.Any()))
		require.NoError(t, err)
		require.Equal(t, ".", setup.Config.Items)
	})
}

func TestRunCleanupResolvesFlagThenConfigThenDisabled(t *testing.T) {
	t.Run("flag overrides configured cleanup", func(t *testing.T) {
		cmd := cmdWithMocks(cmdWithConfig(configWithCleanup()))
		setup, err := cmd.prepareSetup(flagsWithCleanupDisabled(), project.ForProjectInput(project.Any()))
		require.NoError(t, err)
		require.False(t, setup.Config.Cleanup)
	})

	t.Run("configured cleanup used when no flag", func(t *testing.T) {
		cmd := cmdWithMocks(cmdWithConfig(configWithCleanup()))
		setup, err := cmd.prepareSetup(flagsAny(), project.ForProjectInput(project.Any()))
		require.NoError(t, err)
		require.True(t, setup.Config.Cleanup)
	})

	t.Run("cleanup disabled when flag and config unset", func(t *testing.T) {
		cmd := cmdWithMocks()
		setup, err := cmd.prepareSetup(flagsAny(), project.ForProjectInput(project.Any()))
		require.NoError(t, err)
		require.False(t, setup.Config.Cleanup)
	})
}
