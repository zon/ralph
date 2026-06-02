package workflow

import (
	gocontext "context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

var errMockFailure = errors.New("mock failure")

type mockGitHubAuth struct {
	configureFn func(ctx gocontext.Context, owner, repo, secretsDir string) error
}

func (m *mockGitHubAuth) ConfigureGitAuth(ctx gocontext.Context, owner, repo, secretsDir string) error {
	if m.configureFn != nil {
		return m.configureFn(ctx, owner, repo, secretsDir)
	}
	return nil
}

type mockOpenCode struct {
	setupFn func(out *output.Client) error
}

func (m *mockOpenCode) SetupOpenCodeCredentials(out *output.Client) error {
	if m.setupFn != nil {
		return m.setupFn(out)
	}
	return nil
}

type mockGitClient struct {
	configFn             func(global bool, key, value string) error
	fetchBranchFn        func(out *output.Client, branch string) error
	revParseFn           func(args ...string) (string, error)
	mergeBaseFn          func(a, b string) (string, error)
	mergeFn              func(branch string) error
	abortMergeFn         func() error
	tmpPathFn            func(name string) (string, error)
	getCurrentBranchFn   func() (string, error)
	sanitizeBranchNameFn func(name string) string
}

func (m *mockGitClient) Config(global bool, key, value string) error {
	if m.configFn != nil {
		return m.configFn(global, key, value)
	}
	return nil
}

func (m *mockGitClient) FetchBranch(out *output.Client, branch string) error {
	if m.fetchBranchFn != nil {
		return m.fetchBranchFn(out, branch)
	}
	return nil
}

func (m *mockGitClient) RevParse(args ...string) (string, error) {
	if m.revParseFn != nil {
		return m.revParseFn(args...)
	}
	return "abc123", nil
}

func (m *mockGitClient) MergeBase(a, b string) (string, error) {
	if m.mergeBaseFn != nil {
		return m.mergeBaseFn(a, b)
	}
	return "abc123", nil
}

func (m *mockGitClient) Merge(branch string) error {
	if m.mergeFn != nil {
		return m.mergeFn(branch)
	}
	return nil
}

func (m *mockGitClient) AbortMerge() error {
	if m.abortMergeFn != nil {
		return m.abortMergeFn()
	}
	return nil
}

func (m *mockGitClient) TmpPath(name string) (string, error) {
	if m.tmpPathFn != nil {
		return m.tmpPathFn(name)
	}
	return "/tmp/test-" + name, nil
}

func (m *mockGitClient) GetCurrentBranch() (string, error) {
	if m.getCurrentBranchFn != nil {
		return m.getCurrentBranchFn()
	}
	return "feature/test", nil
}

func (m *mockGitClient) SanitizeBranchName(name string) string {
	if m.sanitizeBranchNameFn != nil {
		return m.sanitizeBranchNameFn(name)
	}
	return "test-slug"
}

type mockWorkspace struct {
	prepareFn func(out *output.Client, repoURL, branch, workDir string) error
}

func (m *mockWorkspace) PrepareWorkspace(out *output.Client, repoURL, branch, workDir string) error {
	if m.prepareFn != nil {
		return m.prepareFn(out, repoURL, branch, workDir)
	}
	return nil
}

type mockWorkspaceSetup struct {
	runFn func() error
}

func (m *mockWorkspaceSetup) Run() error {
	if m.runFn != nil {
		return m.runFn()
	}
	return nil
}

type mockConfigLoader struct {
	loadFn func() (*config.RalphConfig, error)
}

func (m *mockConfigLoader) Load() (*config.RalphConfig, error) {
	if m.loadFn != nil {
		return m.loadFn()
	}
	return &config.RalphConfig{MaxIterations: 5}, nil
}

type mockProjectLoader struct {
	loadFn func(path string) (*project.Project, error)
}

func (m *mockProjectLoader) Load(path string) (*project.Project, error) {
	if m.loadFn != nil {
		return m.loadFn(path)
	}
	return &project.Project{Slug: "test-project"}, nil
}

type mockExecutor struct {
	executeFn func(ctx *context.Context, cleanup func(func()), setup *ProjectExecutionSetup) error
}

func (m *mockExecutor) Execute(ctx *context.Context, cleanup func(func()), setup *ProjectExecutionSetup) error {
	if m.executeFn != nil {
		return m.executeFn(ctx, cleanup, setup)
	}
	return nil
}

type deps struct {
	githubAuth     GitHubAuthClient
	openCode       OpenCodeClient
	git            GitClient
	workspace      WorkspaceClient
	workspaceSetup WorkspaceSetupClient
	configLoader   ConfigLoader
	projectLoader  ProjectLoader
	executor       ProjectExecutor
}

type Opt func(*deps)

func withGitHubAuth(c GitHubAuthClient) Opt {
	return func(d *deps) {
		d.githubAuth = c
	}
}

func withOpenCode(c OpenCodeClient) Opt {
	return func(d *deps) {
		d.openCode = c
	}
}

func withGit(c GitClient) Opt {
	return func(d *deps) {
		d.git = c
	}
}

func withWorkspace(c WorkspaceClient) Opt {
	return func(d *deps) {
		d.workspace = c
	}
}

func withWorkspaceSetup(c WorkspaceSetupClient) Opt {
	return func(d *deps) {
		d.workspaceSetup = c
	}
}

func withConfigLoader(c ConfigLoader) Opt {
	return func(d *deps) {
		d.configLoader = c
	}
}

func withProjectLoader(c ProjectLoader) Opt {
	return func(d *deps) {
		d.projectLoader = c
	}
}

func withExecutor(c ProjectExecutor) Opt {
	return func(d *deps) {
		d.executor = c
	}
}

func newWorkflow(opts ...Opt) *Workflow {
	d := &deps{
		githubAuth:     &mockGitHubAuth{},
		openCode:       &mockOpenCode{},
		git:            &mockGitClient{},
		workspace:      &mockWorkspace{},
		workspaceSetup: &mockWorkspaceSetup{},
		configLoader:   &mockConfigLoader{},
		projectLoader:  &mockProjectLoader{},
		executor:       &mockExecutor{},
	}
	for _, opt := range opts {
		opt(d)
	}
	return New(
		d.githubAuth,
		d.openCode,
		d.git,
		d.workspace,
		d.workspaceSetup,
		d.configLoader,
		d.projectLoader,
		d.executor,
	)
}

func testContext() *context.Context {
	ctx := context.NewContext()
	ctx.SetOutput(output.NewClient(io.Discard, io.Discard, false))
	ctx.SetRepo("test-owner/test-repo")
	ctx.SetBranch("feature/test")
	ctx.SetBaseBranch("main")
	ctx.SetProjectFile("/tmp/test-project.yaml")
	ctx.SetBotName("ralph-bot")
	ctx.SetBotEmail("bot@test.com")
	return ctx
}

func TestRun_Success(t *testing.T) {
	executed := false
	w := newWorkflow(
		withExecutor(&mockExecutor{
			executeFn: func(ctx *context.Context, _ func(func()), _ *ProjectExecutionSetup) error {
				executed = true
				return nil
			},
		}),
	)
	err := w.Run(testContext(), nil)
	require.NoError(t, err)
	require.True(t, executed)
}

func TestSetupGitHubAuth_Success(t *testing.T) {
	called := false
	w := newWorkflow(
		withGitHubAuth(&mockGitHubAuth{
			configureFn: func(_ gocontext.Context, owner, repo, secretsDir string) error {
				called = true
				require.Equal(t, "test-owner", owner)
				require.Equal(t, "test-repo", repo)
				require.Equal(t, DefaultSecretsDir, secretsDir)
				return nil
			},
		}),
	)
	err := w.setupGitHubAuth(testContext())
	require.NoError(t, err)
	require.True(t, called)
}

func TestSetupGitHubAuth_EmptyOwnerRepo(t *testing.T) {
	w := newWorkflow()
	ctx := testContext()
	ctx.SetRepoOwner("")
	ctx.SetRepoName("")
	err := w.setupGitHubAuth(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse owner/repo")
}

func TestSetupGitHubAuth_Failure(t *testing.T) {
	w := newWorkflow(
		withGitHubAuth(&mockGitHubAuth{
			configureFn: func(_ gocontext.Context, _, _, _ string) error {
				return errMockFailure
			},
		}),
	)
	err := w.setupGitHubAuth(testContext())
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestConfigureGitUser(t *testing.T) {
	var configKey, configValue string
	w := newWorkflow(
		withGit(&mockGitClient{
			configFn: func(global bool, key, value string) error {
				configKey = key
				configValue = value
				return nil
			},
		}),
	)
	ctx := testContext()
	w.configureGitUser(ctx)
	require.Equal(t, "user.email", configKey)
	require.Equal(t, "bot@test.com", configValue)
}

func TestCloneAndSetupRepo_Success(t *testing.T) {
	workspacePrepared := false
	workspaceSetupRun := false
	w := newWorkflow(
		withWorkspace(&mockWorkspace{
			prepareFn: func(_ *output.Client, repoURL, branch, workDir string) error {
				workspacePrepared = true
				require.Equal(t, "https://github.com/test-owner/test-repo.git", repoURL)
				return nil
			},
		}),
		withWorkspaceSetup(&mockWorkspaceSetup{
			runFn: func() error {
				workspaceSetupRun = true
				return nil
			},
		}),
	)
	err := w.cloneAndSetupRepo(testContext())
	require.NoError(t, err)
	require.True(t, workspacePrepared)
	require.True(t, workspaceSetupRun)
}

func TestCloneAndSetupRepo_WorkspacePrepFails(t *testing.T) {
	w := newWorkflow(
		withWorkspace(&mockWorkspace{
			prepareFn: func(_ *output.Client, _, _, _ string) error {
				return errMockFailure
			},
		}),
	)
	err := w.cloneAndSetupRepo(testContext())
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock failure")
}

func TestCloneAndSetupRepo_SetupWorkspaceFails(t *testing.T) {
	w := newWorkflow(
		withWorkspaceSetup(&mockWorkspaceSetup{
			runFn: func() error {
				return errMockFailure
			},
		}),
	)
	err := w.cloneAndSetupRepo(testContext())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to setup workspace")
}

func TestSyncBaseBranch_FetchFails(t *testing.T) {
	w := newWorkflow(
		withGit(&mockGitClient{
			fetchBranchFn: func(_ *output.Client, _ string) error {
				return errMockFailure
			},
		}),
	)
	err := w.syncBaseBranch(testContext())
	require.NoError(t, err)
}

func TestSyncBaseBranch_NoMergeNeeded(t *testing.T) {
	w := newWorkflow(
		withGit(&mockGitClient{
			revParseFn: func(args ...string) (string, error) {
				return "abc123", nil
			},
			mergeBaseFn: func(a, b string) (string, error) {
				return "abc123", nil
			},
		}),
	)
	err := w.syncBaseBranch(testContext())
	require.NoError(t, err)
}

func TestSyncBaseBranch_MergeNeeded(t *testing.T) {
	w := newWorkflow(
		withGit(&mockGitClient{
			revParseFn: func(args ...string) (string, error) {
				return "def456", nil
			},
			mergeBaseFn: func(a, b string) (string, error) {
				return "abc123", nil
			},
			mergeFn: func(_ string) error {
				return nil
			},
		}),
	)
	err := w.syncBaseBranch(testContext())
	require.NoError(t, err)
}

func TestCheckIfMergeNeeded_BranchDoesNotExist(t *testing.T) {
	w := newWorkflow(
		withGit(&mockGitClient{
			revParseFn: func(args ...string) (string, error) {
				return "", errMockFailure
			},
		}),
	)
	needed, err := w.checkIfMergeNeeded(testContext())
	require.NoError(t, err)
	require.False(t, needed)
}

func TestCheckIfMergeNeeded_MergeBaseError(t *testing.T) {
	w := newWorkflow(
		withGit(&mockGitClient{
			revParseFn: func(args ...string) (string, error) {
				return "def456", nil
			},
			mergeBaseFn: func(a, b string) (string, error) {
				return "", errMockFailure
			},
		}),
	)
	_, err := w.checkIfMergeNeeded(testContext())
	require.Error(t, err)
}

func TestMergeBaseBranch_Success(t *testing.T) {
	merged := false
	w := newWorkflow(
		withGit(&mockGitClient{
			mergeFn: func(_ string) error {
				merged = true
				return nil
			},
		}),
	)
	err := w.mergeBaseBranch(testContext())
	require.NoError(t, err)
	require.True(t, merged)
}

func TestMergeBaseBranch_ConflictTriggersAIResolution(t *testing.T) {
	aborted := false
	tmpFile := ""
	w := newWorkflow(
		withGit(&mockGitClient{
			mergeFn: func(_ string) error {
				return errMockFailure
			},
			abortMergeFn: func() error {
				aborted = true
				return nil
			},
			tmpPathFn: func(name string) (string, error) {
				tmpFile = "/tmp/test-" + name
				return tmpFile, nil
			},
			getCurrentBranchFn: func() (string, error) {
				return "feature/test", nil
			},
			sanitizeBranchNameFn: func(name string) string {
				return "test-slug"
			},
		}),
		withExecutor(&mockExecutor{
			executeFn: func(_ *context.Context, _ func(func()), _ *ProjectExecutionSetup) error {
				return nil
			},
		}),
	)
	defer os.Remove(tmpFile)
	err := w.mergeBaseBranch(testContext())
	require.NoError(t, err)
	require.True(t, aborted)
}

func TestResolveConflictsWithAI_Success(t *testing.T) {
	tmpFile := "/tmp/test-merge-instructions.md"
	executed := false
	w := newWorkflow(
		withGit(&mockGitClient{
			tmpPathFn: func(name string) (string, error) {
				return tmpFile, nil
			},
			getCurrentBranchFn: func() (string, error) {
				return "feature/test", nil
			},
			sanitizeBranchNameFn: func(name string) string {
				return "test-slug"
			},
		}),
		withExecutor(&mockExecutor{
			executeFn: func(_ *context.Context, _ func(func()), _ *ProjectExecutionSetup) error {
				executed = true
				return nil
			},
		}),
	)
	defer os.Remove(tmpFile)
	err := w.resolveConflictsWithAI(testContext())
	require.NoError(t, err)
	require.True(t, executed)
}

func TestResolveConflictsWithAI_TmpPathFails(t *testing.T) {
	w := newWorkflow(
		withGit(&mockGitClient{
			tmpPathFn: func(name string) (string, error) {
				return "", errMockFailure
			},
		}),
	)
	err := w.resolveConflictsWithAI(testContext())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get tmp path")
}

func TestRunProject_Success(t *testing.T) {
	executed := false
	w := newWorkflow(
		withExecutor(&mockExecutor{
			executeFn: func(_ *context.Context, _ func(func()), _ *ProjectExecutionSetup) error {
				executed = true
				return nil
			},
		}),
	)
	err := w.runProject(testContext(), nil)
	require.NoError(t, err)
	require.True(t, executed)
}

func TestPrepareAndExecute_ConfigLoadFails(t *testing.T) {
	w := newWorkflow(
		withConfigLoader(&mockConfigLoader{
			loadFn: func() (*config.RalphConfig, error) {
				return nil, errMockFailure
			},
		}),
	)
	err := w.prepareAndExecute(testContext(), nil, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load config")
}

func TestPrepareAndExecute_ProjectLoadFails(t *testing.T) {
	w := newWorkflow(
		withProjectLoader(&mockProjectLoader{
			loadFn: func(path string) (*project.Project, error) {
				return nil, errMockFailure
			},
		}),
	)
	err := w.prepareAndExecute(testContext(), nil, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load project")
}

func TestPrepareAndExecute_CurrentBranchFails(t *testing.T) {
	w := newWorkflow(
		withGit(&mockGitClient{
			getCurrentBranchFn: func() (string, error) {
				return "", errMockFailure
			},
		}),
	)
	err := w.prepareAndExecute(testContext(), nil, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get current branch")
}

func TestPrepareAndExecute_ExecutesWithInstructions(t *testing.T) {
	var capturedSetup *ProjectExecutionSetup
	w := newWorkflow(
		withExecutor(&mockExecutor{
			executeFn: func(_ *context.Context, _ func(func()), setup *ProjectExecutionSetup) error {
				capturedSetup = setup
				return nil
			},
		}),
	)
	ctx := testContext()
	err := w.prepareAndExecute(ctx, nil, "/tmp/instructions.md")
	require.NoError(t, err)
	require.NotNil(t, capturedSetup)
	require.Equal(t, "test-slug", capturedSetup.BranchName)
}

func TestPrepareAndExecute_ResolvesMaxIterationsFromFlag(t *testing.T) {
	w := newWorkflow()
	ctx := testContext()
	ctx.SetMaxIterations(7)
	err := w.prepareAndExecute(ctx, nil, "")
	require.NoError(t, err)
	require.Equal(t, 7, ctx.MaxIterations())
}

func TestPrepareAndExecute_ResolvesMaxIterationsFromConfig(t *testing.T) {
	w := newWorkflow(
		withConfigLoader(&mockConfigLoader{
			loadFn: func() (*config.RalphConfig, error) {
				return &config.RalphConfig{MaxIterations: 3}, nil
			},
		}),
	)
	ctx := testContext()
	ctx.SetMaxIterations(0)
	err := w.prepareAndExecute(ctx, nil, "")
	require.NoError(t, err)
	require.Equal(t, 3, ctx.MaxIterations())
}

func TestPrepareAndExecute_ResolvesMaxIterationsDefault(t *testing.T) {
	w := newWorkflow(
		withConfigLoader(&mockConfigLoader{
			loadFn: func() (*config.RalphConfig, error) {
				return &config.RalphConfig{}, nil
			},
		}),
	)
	ctx := testContext()
	ctx.SetMaxIterations(0)
	err := w.prepareAndExecute(ctx, nil, "")
	require.NoError(t, err)
	require.Equal(t, 10, ctx.MaxIterations())
}
