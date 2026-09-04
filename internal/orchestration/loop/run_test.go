package loop

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
)

// ---------------------------------------------------------------------------
// Mock types for the RunCmd orchestration
// ---------------------------------------------------------------------------

type mockWorkspaceClient struct {
	ChangeDirectoryFunc func(string) error
	ChangedDirs         []string
}

func (m *mockWorkspaceClient) ChangeDirectory(path string) error {
	m.ChangedDirs = append(m.ChangedDirs, path)
	if m.ChangeDirectoryFunc != nil {
		return m.ChangeDirectoryFunc(path)
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
	return &git.WorktreeCommand{Args: []string{"worktree", "remove", "--force", "/sibling/repo-" + branch}}, nil
}

type mockRemoteRunner struct {
	slug   string
	steps  []string
	max    int
	follow bool
	err    error
	called bool
}

func (m *mockRemoteRunner) Run(slug string, steps []string, max int, follow bool) error {
	m.called = true
	m.slug = slug
	m.steps = steps
	m.max = max
	m.follow = follow
	return m.err
}

// ---------------------------------------------------------------------------
// Builders
// ---------------------------------------------------------------------------

// intPtr returns a pointer to the given int, standing for a --max flag value
// that was explicitly passed on the command line.
func intPtr(v int) *int {
	return &v
}

type runOption func(*RunCmd)

func runWithConfig(cfg config.Loader) runOption {
	return func(c *RunCmd) { c.config = cfg }
}

func runWithLoop(lc *Cmd) runOption {
	return func(c *RunCmd) { c.newLoop = func() (*Cmd, error) { return lc, nil } }
}

func runWithWorktree(wt WorktreeClient) runOption {
	return func(c *RunCmd) { c.worktree = wt }
}

func runWithWorkspace(ws WorkspaceClient) runOption {
	return func(c *RunCmd) { c.workspace = ws }
}

func runWithRemote(remote RemoteRunnerClient) runOption {
	return func(c *RunCmd) { c.remote = remote }
}

func runWithMocks(opts ...runOption) *RunCmd {
	cmd := &RunCmd{
		config:    &config.MockLoader{},
		newLoop:   func() (*Cmd, error) { return loopCmdWithMocks(), nil },
		worktree:  &mockWorktreeClient{},
		workspace: &mockWorkspaceClient{},
		remote:    &mockRemoteRunner{},
	}
	for _, opt := range opts {
		opt(cmd)
	}
	return cmd
}

// loopCmdWithMocks builds an in-process loop Cmd whose every dependency is a
// recording mock, so no test touches the AI, git, or GitHub.
func loopCmdWithMocks() *Cmd {
	return loopCmdWithGit(&mockGitClient{})
}

func loopCmdWithGit(gitClient *mockGitClient) *Cmd {
	return NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{slug: "proposed"},
		&mockAIClient{},
		&mockReportReader{reports: nothingToDoReports()},
		gitClient,
		&mockPullRequestOpener{},
		envNotInWorkflow(),
	)
}

func loopFlagsAny() LoopFlags {
	return LoopFlags{Slug: "fmt", Max: intPtr(10)}
}

func loopFlagsWithMode(mode string) LoopFlags {
	return LoopFlags{Slug: "fmt", Max: intPtr(10), Mode: mode}
}

func loopFlagsWithFollow() LoopFlags {
	return LoopFlags{Slug: "fmt", Max: intPtr(10), Follow: true}
}

func loopFlagsWithFollowAndMode(mode string) LoopFlags {
	return LoopFlags{Slug: "fmt", Max: intPtr(10), Follow: true, Mode: mode}
}

func configWithMode(mode string) config.Loader {
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) { return config.WithMode(mode), nil },
	}
}

// configWithLoopMax returns a loader whose config has a loops entry for the
// slug carrying the given max iteration cap.
func configWithLoopMax(slug string, max int) config.Loader {
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) {
			cfg := config.Any()
			m := max
			cfg.Loops = []config.LoopConfig{{Slug: slug, Steps: []string{"run gofmt"}, Max: &m}}
			return cfg, nil
		},
	}
}

// configWithLoopMaxNoMax returns a loader whose config has a loops entry for
// the slug without a max field.
func configWithLoopMaxNoMax(slug string) config.Loader {
	return &config.MockLoader{
		LoadFn: func() (*config.RalphConfig, error) {
			cfg := config.Any()
			cfg.Loops = []config.LoopConfig{{Slug: slug, Steps: []string{"run gofmt"}}}
			return cfg, nil
		},
	}
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func worktreeCreated(cmd *RunCmd) bool {
	if m, ok := cmd.worktree.(*mockWorktreeClient); ok {
		return m.CreateWorktreeCalled
	}
	return false
}

func worktreeRemoved(cmd *RunCmd) bool {
	if m, ok := cmd.worktree.(*mockWorktreeClient); ok {
		return m.RemoveWorktreeCalled
	}
	return false
}

func worktreeCreateBranch(cmd *RunCmd) string {
	if m, ok := cmd.worktree.(*mockWorktreeClient); ok {
		return m.CreateWorktreeBranch
	}
	return ""
}

// ---------------------------------------------------------------------------
// Tests: mode resolution
// ---------------------------------------------------------------------------

func TestRunModeResolvesFlagThenConfigThenLocal(t *testing.T) {
	t.Run("flag overrides configured mode", func(t *testing.T) {
		cmd := runWithMocks(runWithConfig(configWithMode(config.ModeRemote)))
		result, err := cmd.Run(loopFlagsWithMode(config.ModeLocal))
		require.NoError(t, err)
		require.NotNil(t, result, "local mode resolves the loop in-process")
		require.Equal(t, "fmt", result.Slug)
		require.False(t, cmd.remote.(*mockRemoteRunner).called, "remote is not consulted when the flag says local")
		require.False(t, worktreeCreated(cmd), "no worktree is created when the flag says local")
	})

	t.Run("configured mode used when no flag", func(t *testing.T) {
		cmd := runWithMocks(runWithConfig(configWithMode(config.ModeRemote)))
		result, err := cmd.Run(loopFlagsAny())
		require.NoError(t, err)
		require.Nil(t, result, "remote mode returns no in-process resolution")
		require.True(t, cmd.remote.(*mockRemoteRunner).called, "remote is consulted when the config says remote")
	})

	t.Run("local default when flag and config unset", func(t *testing.T) {
		cmd := runWithMocks()
		result, err := cmd.Run(loopFlagsAny())
		require.NoError(t, err)
		require.NotNil(t, result, "local mode resolves the loop in-process")
		require.False(t, worktreeCreated(cmd), "the local default creates no worktree")
		require.False(t, cmd.remote.(*mockRemoteRunner).called, "remote is not consulted for the local default")
	})
}

func TestRunInvalidModeRejectedBeforeExecution(t *testing.T) {
	cmd := runWithMocks()
	_, err := cmd.Run(loopFlagsWithMode("sandbox"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid mode: sandbox (expected local, worktree, or remote)")
	require.False(t, cmd.remote.(*mockRemoteRunner).called)
	require.False(t, worktreeCreated(cmd))
}

func TestRunInvalidConfiguredModeRejectedBeforeExecution(t *testing.T) {
	cmd := runWithMocks(runWithConfig(configWithMode("sandbox")))
	_, err := cmd.Run(loopFlagsAny())
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid mode: sandbox (expected local, worktree, or remote)")
	require.False(t, cmd.remote.(*mockRemoteRunner).called)
	require.False(t, worktreeCreated(cmd))
}

// ---------------------------------------------------------------------------
// Tests: --follow rejection and acceptance
// ---------------------------------------------------------------------------

func TestRunFollowRejectedForLocalAndWorktreeModes(t *testing.T) {
	for _, mode := range []string{config.ModeLocal, config.ModeWorktree} {
		t.Run(mode, func(t *testing.T) {
			cmd := runWithMocks()
			_, err := cmd.Run(loopFlagsWithFollowAndMode(mode))
			require.Error(t, err)
			require.Contains(t, err.Error(), "--follow flag is not applicable with --mode "+mode)
			require.False(t, cmd.remote.(*mockRemoteRunner).called)
			require.False(t, worktreeCreated(cmd))
		})
	}
}

func TestRunFollowRejectedWhenWorktreeFromConfig(t *testing.T) {
	cmd := runWithMocks(runWithConfig(configWithMode(config.ModeWorktree)))
	_, err := cmd.Run(loopFlagsWithFollow())
	require.Error(t, err)
	require.Contains(t, err.Error(), "--follow flag is not applicable with --mode worktree")
	require.False(t, worktreeCreated(cmd))
}

func TestRunFollowRejectedAgainstLocalDefault(t *testing.T) {
	cmd := runWithMocks()
	_, err := cmd.Run(loopFlagsWithFollow())
	require.Error(t, err)
	require.Contains(t, err.Error(), "--follow flag is not applicable with --mode local")
	require.False(t, worktreeCreated(cmd))
	require.False(t, cmd.remote.(*mockRemoteRunner).called)
}

func TestRunFollowAcceptedForRemoteMode(t *testing.T) {
	cmd := runWithMocks()
	_, err := cmd.Run(loopFlagsWithFollowAndMode(config.ModeRemote))
	require.NoError(t, err)
	require.True(t, cmd.remote.(*mockRemoteRunner).called, "remote is consulted with --follow")
	require.True(t, cmd.remote.(*mockRemoteRunner).follow, "the --follow flag is passed through")
}

// ---------------------------------------------------------------------------
// Tests: mode dispatch
// ---------------------------------------------------------------------------

func TestRunLocalDispatchesToInProcessLoop(t *testing.T) {
	gitClient := &mockGitClient{}
	ai := &mockAIClient{}
	lc := NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{slug: "proposed"},
		ai,
		&mockReportReader{reports: nothingToDoReports()},
		gitClient,
		&mockPullRequestOpener{},
		envNotInWorkflow(),
	)
	cmd := runWithMocks(runWithLoop(lc))

	result, err := cmd.Run(loopFlagsWithMode(config.ModeLocal))
	require.NoError(t, err)
	require.NotNil(t, result, "local mode resolves the loop in-process")
	require.Equal(t, "fmt", result.Slug)
	require.Equal(t, []string{"run gofmt"}, result.Steps)
	require.Equal(t, 1, ai.calls, "the loop runs in-process in local mode")
	require.Equal(t, 1, gitClient.switchCalls, "local mode switches to the loop branch before iterating")
	require.False(t, worktreeCreated(cmd), "local mode creates no worktree")
	require.False(t, cmd.remote.(*mockRemoteRunner).called, "local mode does not consult remote")
}

func TestRunRemoteDispatchesToRemoteRunner(t *testing.T) {
	cmd := runWithMocks()
	result, err := cmd.Run(loopFlagsWithMode(config.ModeRemote))
	require.NoError(t, err)
	require.Nil(t, result, "remote mode returns no in-process resolution")
	require.True(t, cmd.remote.(*mockRemoteRunner).called)
	require.Equal(t, "fmt", cmd.remote.(*mockRemoteRunner).slug)
	require.Equal(t, 10, cmd.remote.(*mockRemoteRunner).max)
}

func TestRunRemotePropagatesSubmitError(t *testing.T) {
	submitErr := errors.New("failed to submit workflow: boom")
	runner := &mockRemoteRunner{err: submitErr}
	cmd := runWithMocks(runWithRemote(runner))

	_, err := cmd.Run(loopFlagsWithMode(config.ModeRemote))
	require.Error(t, err)
	require.Equal(t, submitErr, err)
	require.True(t, runner.called)
}

// ---------------------------------------------------------------------------
// Tests: iteration cap resolution
// ---------------------------------------------------------------------------

// TestRunLoopMaxResolution asserts the iteration cap follows the three-level
// precedence: --max when passed, otherwise the matching loop config entry's
// max field, otherwise the default of 20. Remote mode surfaces the resolved
// cap on the workflow submission.
func TestRunLoopMaxResolution(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Loader
		flags   LoopFlags
		wantMax int
	}{
		{
			name:    "flag overrides the config entry's max",
			config:  configWithLoopMax("fmt", 30),
			flags:   LoopFlags{Slug: "fmt", Max: intPtr(5), Mode: config.ModeRemote},
			wantMax: 5,
		},
		{
			name:    "config entry's max used when no flag is passed",
			config:  configWithLoopMax("fmt", 30),
			flags:   LoopFlags{Slug: "fmt", Mode: config.ModeRemote},
			wantMax: 30,
		},
		{
			name:    "default of 20 when the config entry sets no max",
			config:  configWithLoopMaxNoMax("fmt"),
			flags:   LoopFlags{Slug: "fmt", Mode: config.ModeRemote},
			wantMax: 20,
		},
		{
			name:    "default of 20 when no entry matches the slug",
			config:  configWithLoopMax("other", 30),
			flags:   LoopFlags{Slug: "fmt", Mode: config.ModeRemote},
			wantMax: 20,
		},
		{
			name:    "default of 20 when the config has no loops",
			config:  &config.MockLoader{},
			flags:   LoopFlags{Slug: "fmt", Mode: config.ModeRemote},
			wantMax: 20,
		},
		{
			name:    "steps without a slug ignore every config entry's max",
			config:  configWithLoopMax("fmt", 30),
			flags:   LoopFlags{Steps: []string{"run gofmt"}, Mode: config.ModeRemote},
			wantMax: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := runWithMocks(runWithConfig(tt.config))
			_, err := cmd.Run(tt.flags)
			require.NoError(t, err)
			require.Equal(t, tt.wantMax, cmd.remote.(*mockRemoteRunner).max, "the workflow submission carries the resolved cap")
		})
	}
}

// TestRunLoopMaxConfigEntryCapsLocalIterations asserts local mode runs the
// loop at most the loop config entry's max iterations when --max is not
// passed.
func TestRunLoopMaxConfigEntryCapsLocalIterations(t *testing.T) {
	ai := &mockAIClient{}
	lc := NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{slug: "proposed"},
		ai,
		&mockReportReader{reports: []string{"did the work"}},
		&mockGitClient{},
		&mockPullRequestOpener{},
		envNotInWorkflow(),
	)
	cmd := runWithMocks(runWithConfig(configWithLoopMax("fmt", 3)), runWithLoop(lc))

	result, err := cmd.Run(LoopFlags{Slug: "fmt", Mode: config.ModeLocal})
	require.NoError(t, err)
	require.NotNil(t, result, "local mode resolves the loop in-process")
	require.Equal(t, 3, ai.calls, "the loop runs exactly the config entry's max iterations")
}

// TestRunLoopMaxFlagCapsLocalIterations asserts an explicitly passed --max
// caps local iterations ahead of the loop config entry's max field.
func TestRunLoopMaxFlagCapsLocalIterations(t *testing.T) {
	ai := &mockAIClient{}
	lc := NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{slug: "proposed"},
		ai,
		&mockReportReader{reports: []string{"did the work"}},
		&mockGitClient{},
		&mockPullRequestOpener{},
		envNotInWorkflow(),
	)
	cmd := runWithMocks(runWithConfig(configWithLoopMax("fmt", 30)), runWithLoop(lc))

	result, err := cmd.Run(LoopFlags{Slug: "fmt", Max: intPtr(3), Mode: config.ModeLocal})
	require.NoError(t, err)
	require.NotNil(t, result, "local mode resolves the loop in-process")
	require.Equal(t, 3, ai.calls, "the loop runs exactly --max iterations, ahead of the config entry's max")
}

// ---------------------------------------------------------------------------
// Tests: worktree mode
// ---------------------------------------------------------------------------

func TestRunWorktreeCreatesWorktreeOnLoopBranchRunsLoopAndRemovesIt(t *testing.T) {
	ws := &mockWorkspaceClient{}
	wt := &mockWorktreeClient{}
	gitClient := &mockGitClient{}
	ai := &mockAIClient{}
	lc := NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{slug: "proposed"},
		ai,
		&mockReportReader{reports: nothingToDoReports()},
		gitClient,
		&mockPullRequestOpener{},
		envNotInWorkflow(),
	)
	cmd := runWithMocks(runWithWorkspace(ws), runWithWorktree(wt), runWithLoop(lc))

	result, err := cmd.Run(loopFlagsWithMode(config.ModeWorktree))
	require.NoError(t, err)
	require.NotNil(t, result, "worktree mode resolves the loop in-process")
	require.Equal(t, "fmt", result.Slug)
	require.True(t, wt.BranchCheckedOutInWorktreeCalled, "the loop branch is checked before the worktree is created")
	require.True(t, wt.CreateWorktreeCalled, "a worktree is created for the loop branch")
	require.Equal(t, "loop-fmt", wt.CreateWorktreeBranch, "the worktree is created on the loop-<slug> branch")
	require.Equal(t, 1, ai.calls, "the loop runs in-process inside the worktree")
	require.Zero(t, gitClient.switchCalls, "the branch switch is skipped inside the worktree")
	require.True(t, wt.RemoveWorktreeCalled, "the worktree is removed when the loop ends")
	require.Equal(t, "loop-fmt", wt.RemoveWorktreeBranch)
}

func TestRunWorktreeRunsLoopInsideWorktreeDirectory(t *testing.T) {
	var events []string
	ws := &mockWorkspaceClient{
		ChangeDirectoryFunc: func(dir string) error {
			if dir == "/sibling/repo-loop-fmt" {
				events = append(events, "entered-worktree")
			}
			return nil
		},
	}
	lc := NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{slug: "proposed"},
		&eventAIClient{events: &events},
		&mockReportReader{reports: nothingToDoReports()},
		&mockGitClient{},
		&mockPullRequestOpener{},
		envNotInWorkflow(),
	)
	cmd := runWithMocks(runWithWorkspace(ws), runWithLoop(lc))

	_, err := cmd.Run(loopFlagsWithMode(config.ModeWorktree))
	require.NoError(t, err)
	require.Equal(t, []string{"entered-worktree", "ran-loop"}, events, "the loop runs after changing into the worktree")
}

func TestRunWorktreeReturnsToMainCheckoutBeforeRemoval(t *testing.T) {
	ws := &mockWorkspaceClient{}
	cmd := runWithMocks(runWithWorkspace(ws))
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	_, err = cmd.Run(loopFlagsWithMode(config.ModeWorktree))
	require.NoError(t, err)

	require.Equal(t, []string{"/sibling/repo-loop-fmt", originalDir}, ws.ChangedDirs,
		"the run changes back out of the worktree before removing it")
}

func TestRunWorktreeRemovesWorktreeWhenLoopFails(t *testing.T) {
	aiErr := errors.New("opencode execution failed: boom")
	ai := &mockAIClient{err: aiErr}
	lc := NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{slug: "proposed"},
		ai,
		&mockReportReader{reports: nothingToDoReports()},
		&mockGitClient{},
		&mockPullRequestOpener{},
		envNotInWorkflow(),
	)
	wt := &mockWorktreeClient{}
	cmd := runWithMocks(runWithWorktree(wt), runWithLoop(lc))

	_, err := cmd.Run(loopFlagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Equal(t, aiErr, err, "the loop's error is returned")
	require.True(t, worktreeRemoved(cmd), "the worktree is removed when the loop fails")
}

func TestRunWorktreeRemovalErrorReportedWhenRunSucceeds(t *testing.T) {
	wt := &mockWorktreeClient{
		RemoveWorktreeFunc: func(string, bool) (*git.WorktreeCommand, error) {
			return nil, errors.New("failed to remove worktree")
		},
	}
	cmd := runWithMocks(runWithWorktree(wt))

	_, err := cmd.Run(loopFlagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to remove worktree")
}

func TestRunWorktreeLoopErrorTakesPrecedenceOverRemovalError(t *testing.T) {
	aiErr := errors.New("iteration failed")
	ai := &mockAIClient{err: aiErr}
	lc := NewCmd(
		&mockLoopConfigClient{loops: map[string][]string{"fmt": {"run gofmt"}}},
		&mockPromptBuilder{},
		&mockSlugProposer{slug: "proposed"},
		ai,
		&mockReportReader{reports: nothingToDoReports()},
		&mockGitClient{},
		&mockPullRequestOpener{},
		envNotInWorkflow(),
	)
	wt := &mockWorktreeClient{
		RemoveWorktreeFunc: func(string, bool) (*git.WorktreeCommand, error) {
			return nil, errors.New("failed to remove worktree")
		},
	}
	cmd := runWithMocks(runWithWorktree(wt), runWithLoop(lc))

	_, err := cmd.Run(loopFlagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "iteration failed", "the loop's error is reported, not the removal error")
}

func TestRunWorktreeBranchCheckedOutElsewhereReturnsErrorAndCreatesNothing(t *testing.T) {
	wt := &mockWorktreeClient{checkedOut: true}
	cmd := runWithMocks(runWithWorktree(wt))

	_, err := cmd.Run(loopFlagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "branch 'loop-fmt' is already checked out in another worktree")
	require.False(t, worktreeCreated(cmd), "no worktree is created when the loop branch is already checked out")
	require.False(t, worktreeRemoved(cmd))
	require.False(t, cmd.remote.(*mockRemoteRunner).called)
}

func TestRunWorktreeListFailureAbortsBeforeCreate(t *testing.T) {
	wt := &mockWorktreeClient{
		BranchCheckedOutInWorktreeFunc: func(string, bool) (*git.WorktreeCommand, bool, error) {
			return nil, false, errors.New("failed to list worktrees")
		},
	}
	cmd := runWithMocks(runWithWorktree(wt))

	_, err := cmd.Run(loopFlagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to list worktrees")
	require.False(t, worktreeCreated(cmd))
	require.False(t, worktreeRemoved(cmd))
}

func TestRunWorktreeCreateFailureAbortsWithoutRemoval(t *testing.T) {
	wt := &mockWorktreeClient{
		CreateWorktreeFunc: func(string, bool) (*git.WorktreeCommand, error) {
			return nil, errors.New("failed to create worktree")
		},
	}
	cmd := runWithMocks(runWithWorktree(wt))

	_, err := cmd.Run(loopFlagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create worktree")
	require.False(t, worktreeRemoved(cmd), "nothing is removed when no worktree was created")
}

func TestRunWorktreeResolvesSlugBeforeCreatingWorktree(t *testing.T) {
	proposer := &mockSlugProposer{slug: "gofmt"}
	lc := NewCmd(
		&mockLoopConfigClient{},
		&mockPromptBuilder{},
		proposer,
		&mockAIClient{},
		&mockReportReader{reports: nothingToDoReports()},
		&mockGitClient{},
		&mockPullRequestOpener{},
		envNotInWorkflow(),
	)
	wt := &mockWorktreeClient{}
	cmd := runWithMocks(runWithWorktree(wt), runWithLoop(lc))

	result, err := cmd.Run(LoopFlags{Steps: []string{"run gofmt"}, Max: intPtr(10), Mode: config.ModeWorktree})
	require.NoError(t, err)
	require.True(t, proposer.called, "the slug proposer is consulted before the worktree is created")
	require.Equal(t, "gofmt", result.Slug, "the proposed slug is resolved")
	require.Equal(t, "loop-gofmt", wt.CreateWorktreeBranch, "the worktree is created on the proposed slug's loop branch")
}

// eventAIClient records each agent pass as a "ran-loop" event, so worktree
// tests can assert the loop runs after the process changes directory.
type eventAIClient struct {
	events *[]string
}

func (e *eventAIClient) RunAgent(prompt string) error {
	*e.events = append(*e.events, "ran-loop")
	return nil
}

func (e *eventAIClient) PrintStats() {}
