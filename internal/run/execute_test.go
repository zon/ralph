package run

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func TestExecute_NonExistentProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "non-existent.yaml")

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))

	setup, err := PrepareExecution(ctx)
	assert.Error(t, err, "PrepareExecution should return error when project file does not exist")
	assert.Nil(t, setup)
}

func TestExecute_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "invalid.yaml")
	invalidYAML := "name: Test\ndescription: [invalid yaml structure\nrequirements:\n  - not properly formatted\n"
	require.NoError(t, os.WriteFile(projectFile, []byte(invalidYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	setup, err := PrepareExecution(ctx)

	require.Error(t, err)
	assert.Nil(t, setup)
}

func TestExecute_EmptyRequirements(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "empty-reqs.yaml")
	emptyReqsYAML := "name: Test Project\ndescription: Project with no requirements\nrequirements: []\n"
	require.NoError(t, os.WriteFile(projectFile, []byte(emptyReqsYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	setup, err := PrepareExecution(ctx)

	require.Error(t, err)
	assert.Nil(t, setup)
}

// projectYAML is a minimal valid project used across development iteration tests.
const projectYAML = `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

func TestExecute_ValidProject(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := project.ExecuteDevelopmentIteration(ctx, nil)

	require.NoError(t, err)
}

func TestExecute_BlockedMDExists(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))
	require.NoError(t, os.WriteFile("blocked.md", []byte("Agent is blocked due to previous error"), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := project.ExecuteDevelopmentIteration(ctx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestExecute_BlockedMDContents(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))
	blockedContent := "This is the blocked reason from blocked.md"
	require.NoError(t, os.WriteFile("blocked.md", []byte(blockedContent), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := project.ExecuteDevelopmentIteration(ctx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), blockedContent)
}

func TestExecute_NoBlockedMD(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := project.ExecuteDevelopmentIteration(ctx, nil)

	require.NoError(t, err)
}

func TestExecute_NormalizeTrailingNewlines(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	withExcessNewlines := projectYAML + "\n\n\n"
	require.NoError(t, os.WriteFile("test-project.yaml", []byte(withExcessNewlines), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := project.ExecuteDevelopmentIteration(ctx, nil)
	require.NoError(t, err)

	content, err := os.ReadFile("test-project.yaml")
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(string(content), "\n"), "file should end with a newline")
	assert.False(t, strings.HasSuffix(string(content), "\n\n"), "file should have exactly one trailing newline")
}

func TestExecute_StagesFileWithChanges(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	// Commit a project file with excess trailing newlines so normalization produces a tracked change.
	withExcessNewlines := projectYAML + "\n\n\n"
	require.NoError(t, os.WriteFile("test-project.yaml", []byte(withExcessNewlines), 0644))
	for _, args := range [][]string{
		{"add", "test-project.yaml"},
		{"commit", "-m", "add project file"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		require.NoError(t, c.Run())
	}

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := project.ExecuteDevelopmentIteration(ctx, nil)
	require.NoError(t, err)

	// Normalization should have changed and staged the file.
	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	cmd.Dir = workDir
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "test-project.yaml")
}

func TestExecute_DoesNotStageFileWithoutChanges(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	// Write the project file but do not commit it. Untracked files return no
	// diff against the index, so HasFileChanges returns false and StageFile is
	// never called regardless of mock-AI modifications.
	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := project.ExecuteDevelopmentIteration(ctx, nil)
	require.NoError(t, err)

	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	cmd.Dir = workDir
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.NotContains(t, string(out), "test-project.yaml")
}

func TestExecute_StartsServices(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	// Append a service to the existing .ralph/config.yaml.
	existing, err := os.ReadFile(filepath.Join(workDir, ".ralph", "config.yaml"))
	require.NoError(t, err)
	servicesCfg := string(existing) + "services:\n  - name: test-service\n    command: echo\n    args:\n      - test\n"
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ".ralph", "config.yaml"), []byte(servicesCfg), 0644))

	ctx := testutil.NewContext(
		testutil.WithProjectFile("test-project.yaml"),
		testutil.WithNoServices(false),
	)
	err = project.ExecuteDevelopmentIteration(ctx, nil)

	require.NoError(t, err)
}

func TestExecute_WritesBlockedOnAgentFailure(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	t.Setenv("RALPH_MOCK_AI_FAIL", "true")
	logger.SetVerbose(true)

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := project.ExecuteDevelopmentIteration(ctx, nil)

	require.Error(t, err)

	blockedContent, err := os.ReadFile(filepath.Join(workDir, "blocked.md"))
	require.NoError(t, err)
	assert.Contains(t, string(blockedContent), "Blocked")
	assert.Contains(t, string(blockedContent), "opencode execution failed")
}

func TestExecute_PropagatesAdapterErrors(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	adapter := &mockAdapter{
		RunBeforeCommandsFunc: func(cfg *config.RalphConfig) error {
			return errors.New("adapter error")
		},
	}

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	ctx.SetWorkflowExecution(true)

	setup := &ExecutionSetup{
		ProjectFile: "test-project.yaml",
		Project:     mustLoadProject("test-project.yaml"),
		Config:      mustLoadConfig(),
		BranchName:  "test-project",
		BaseBranch:  "main",
	}

	err := Execute(ctx, nil, setup, adapter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adapter error")
}

func TestExecute_DefaultAdapterUsedWhenNil(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	adapter := &mockAdapter{
		RunBeforeCommandsFunc: func(cfg *config.RalphConfig) error {
			return errors.New("simulated error")
		},
	}

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	ctx.SetWorkflowExecution(true)

	setup := &ExecutionSetup{
		ProjectFile: "test-project.yaml",
		Project:     mustLoadProject("test-project.yaml"),
		Config:      mustLoadConfig(),
		BranchName:  "test-project",
		BaseBranch:  "main",
	}

	err := Execute(ctx, nil, setup, adapter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated error")
}

func TestExecute_AdapterNotifiesErrorOnFailure(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	t.Setenv("RALPH_MOCK_AI_FAIL", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	notifyErrorCalled := false
	adapter := &mockAdapter{
		RunBeforeCommandsFunc: func(cfg *config.RalphConfig) error {
			return nil
		},
		NotifyErrorFunc: func(projectName string, shouldNotify bool) {
			notifyErrorCalled = true
		},
		NotifySuccessFunc: func(projectName string, shouldNotify bool) {},
		LogVerboseFunc:    func(format string, args ...interface{}) {},
	}

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	ctx.SetWorkflowExecution(true)

	setup := &ExecutionSetup{
		ProjectFile: "test-project.yaml",
		Project:     mustLoadProject("test-project.yaml"),
		Config:      mustLoadConfig(),
		BranchName:  "test-project",
		BaseBranch:  "main",
	}

	err := Execute(ctx, nil, setup, adapter)
	require.Error(t, err)
	assert.True(t, notifyErrorCalled, "NotifyError should be called when iteration loop fails")
}

type mockAdapter struct {
	RunBeforeCommandsFunc func(*config.RalphConfig) error
	GetCommitLogFunc      func(string, int) (string, error)
	NotifyErrorFunc       func(string, bool)
	NotifySuccessFunc     func(string, bool)
	CreatePullRequestFunc func(*context.Context, *project.Project, string, string, string) (string, error)
	GeneratePRSummaryFunc func(*context.Context, string, string, string, string) (string, error)
	LogVerboseFunc        func(string, ...interface{})
	LogVerboseFnFunc      func(func() string)
	LogSuccessFunc        func(string, ...interface{})
}

func (m *mockAdapter) RunBeforeCommands(cfg *config.RalphConfig) error {
	if m.RunBeforeCommandsFunc != nil {
		return m.RunBeforeCommandsFunc(cfg)
	}
	return nil
}

func (m *mockAdapter) GetCommitLog(baseBranch string, n int) (string, error) {
	if m.GetCommitLogFunc != nil {
		return m.GetCommitLogFunc(baseBranch, n)
	}
	return "", nil
}

func (m *mockAdapter) NotifyError(projectName string, shouldNotify bool) {
	if m.NotifyErrorFunc != nil {
		m.NotifyErrorFunc(projectName, shouldNotify)
	}
}

func (m *mockAdapter) NotifySuccess(projectName string, shouldNotify bool) {
	if m.NotifySuccessFunc != nil {
		m.NotifySuccessFunc(projectName, shouldNotify)
	}
}

func (m *mockAdapter) CreatePullRequest(ctx *context.Context, proj *project.Project, branchName, baseBranch, prSummary string) (string, error) {
	if m.CreatePullRequestFunc != nil {
		return m.CreatePullRequestFunc(ctx, proj, branchName, baseBranch, prSummary)
	}
	return "", nil
}

func (m *mockAdapter) GeneratePRSummary(ctx *context.Context, projectDesc, projectStatus, baseBranch, commitLog string) (string, error) {
	if m.GeneratePRSummaryFunc != nil {
		return m.GeneratePRSummaryFunc(ctx, projectDesc, projectStatus, baseBranch, commitLog)
	}
	return "", nil
}

func (m *mockAdapter) LogVerbose(format string, args ...interface{}) {
	if m.LogVerboseFunc != nil {
		m.LogVerboseFunc(format, args...)
	}
}

func (m *mockAdapter) LogVerboseFn(fn func() string) {
	if m.LogVerboseFnFunc != nil {
		m.LogVerboseFnFunc(fn)
	}
}

func (m *mockAdapter) LogSuccess(format string, args ...interface{}) {
	if m.LogSuccessFunc != nil {
		m.LogSuccessFunc(format, args...)
	}
}

func mustLoadProject(path string) *project.Project {
	proj, err := project.LoadProject(path)
	if err != nil {
		panic(err)
	}
	return proj
}

func mustLoadConfig() *config.RalphConfig {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	return cfg
}
