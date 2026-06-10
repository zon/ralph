package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orchestrationWorkflow "github.com/zon/ralph/internal/orchestration/workflowrun"
	"github.com/zon/ralph/internal/orchestration/workspace"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/config"
)

func TestGitAdapterFetchBranch_NoRemoteReturnsError(t *testing.T) {
	t.Parallel()

	adapter := &gitAdapter{}
	err := adapter.FetchBranch("nonexistent-branch")
	require.Error(t, err)
}

func TestGitAdapterNeedsMerge_NoBranchReturnsFalse(t *testing.T) {
	t.Parallel()

	adapter := &gitAdapter{}
	needs, err := adapter.NeedsMerge("nonexistent-branch")
	require.NoError(t, err)
	assert.False(t, needs)
}

func TestGitAdapterAbortMerge_NoMergeInProgressDoesNotPanic(t *testing.T) {
	t.Parallel()

	adapter := &gitAdapter{}
	adapter.AbortMerge()
}

// ---------------------------------------------------------------------------
// Mocks for orchestration WorkflowRunCmd tests
// ---------------------------------------------------------------------------

type mockWorWorkspaceSetupClient struct {
	setupFn func(flags workspace.WorkspaceFlags) error
}

func (m *mockWorWorkspaceSetupClient) Setup(flags workspace.WorkspaceFlags) error {
	if m.setupFn != nil {
		return m.setupFn(flags)
	}
	return nil
}

type mockWorGitClient struct {
	fetchBranchFn func(branch string) error
	needsMergeFn  func(branch string) (bool, error)
	mergeFn       func(branch string) error
	abortMergeFn  func()
}

func (m *mockWorGitClient) FetchBranch(branch string) error {
	if m.fetchBranchFn != nil {
		return m.fetchBranchFn(branch)
	}
	return nil
}

func (m *mockWorGitClient) NeedsMerge(branch string) (bool, error) {
	if m.needsMergeFn != nil {
		return m.needsMergeFn(branch)
	}
	return false, nil
}

func (m *mockWorGitClient) Merge(branch string) error {
	if m.mergeFn != nil {
		return m.mergeFn(branch)
	}
	return nil
}

func (m *mockWorGitClient) AbortMerge() {
	if m.abortMergeFn != nil {
		m.abortMergeFn()
	}
}

type mockWorAIClient struct {
	resolveMergeConflictsFn func(baseBranch, projectBranch string) error
}

func (m *mockWorAIClient) ResolveMergeConflicts(baseBranch, projectBranch string) error {
	if m.resolveMergeConflictsFn != nil {
		return m.resolveMergeConflictsFn(baseBranch, projectBranch)
	}
	return nil
}

type mockWorRunnerClient struct {
	runLocalFn func(proj *project.Project, cfg *config.RalphConfig) error
}

func (m *mockWorRunnerClient) RunLocal(proj *project.Project, cfg *config.RalphConfig) error {
	if m.runLocalFn != nil {
		return m.runLocalFn(proj, cfg)
	}
	return nil
}

type mockWorConfigClient struct {
	loadOptionalFn func() (*config.RalphConfig, error)
}

func (m *mockWorConfigClient) LoadOptional() (*config.RalphConfig, error) {
	if m.loadOptionalFn != nil {
		return m.loadOptionalFn()
	}
	return &config.RalphConfig{}, nil
}

type mockWorProjectClient struct {
	loadFn func(path string) (*project.Project, error)
}

func (m *mockWorProjectClient) Load(path string) (*project.Project, error) {
	if m.loadFn != nil {
		return m.loadFn(path)
	}
	return &project.Project{}, nil
}

type mockWorDebugClient struct {
	setupFn func(branch string) error
}

func (m *mockWorDebugClient) Setup(branch string) error {
	if m.setupFn != nil {
		return m.setupFn(branch)
	}
	return nil
}

type mockWorOutputClient struct {
	warnfFn func(format string, a ...any)
}

func (m *mockWorOutputClient) Warnf(format string, a ...any) {
	if m.warnfFn != nil {
		m.warnfFn(format, a...)
	}
}

// ---------------------------------------------------------------------------
// Tests for orchestration WorkflowRunCmd.Run
// ---------------------------------------------------------------------------

func TestWorkflowRunCmd_MissingProjectPath(t *testing.T) {
	t.Parallel()

	cmd := orchestrationWorkflow.NewWorkflowRunCmd(
		&mockWorWorkspaceSetupClient{},
		&mockWorGitClient{},
		&mockWorAIClient{},
		&mockWorRunnerClient{},
		&mockWorConfigClient{},
		&mockWorProjectClient{},
		&mockWorDebugClient{},
		&mockWorOutputClient{},
	)

	flags := orchestrationWorkflow.WorkflowRunFlags{
		ProjectPath: "",
	}
	err := cmd.Run(flags)
	require.Error(t, err)
	assert.ErrorIs(t, err, orchestrationWorkflow.ErrMissingProjectPath)
}

func TestWorkflowRunCmd_FlagPropagation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		flags        orchestrationWorkflow.WorkflowRunFlags
		debugBranch  string
		wantBase     string
		wantIter     int
		wantModel    string
	}{
		{
			name: "propagates all flags to orchestration",
			flags: orchestrationWorkflow.WorkflowRunFlags{
				ProjectPath:    "test.yaml",
				Repo:           "owner/repo",
				CloneBranch:    "main",
				BaseBranch:     "base-branch",
				ProjectBranch:  "feature",
				BotName:        "bot",
				BotEmail:       "bot@test.com",
				MaxIterations:  5,
				InstructionsMd: "custom instructions",
				Model:          "gpt-4",
				NoServices:     true,
			},
			wantBase:  "base-branch",
			wantIter:  5,
			wantModel: "gpt-4",
		},
		{
			name: "default values when flags are empty",
			flags: orchestrationWorkflow.WorkflowRunFlags{
				ProjectPath: "test.yaml",
			},
			wantBase:  "",
			wantIter:  0,
			wantModel: "",
		},
		{
			name: "sets debug when provided",
			flags: orchestrationWorkflow.WorkflowRunFlags{
				ProjectPath: "test.yaml",
				Debug:       "fix-bug",
			},
			debugBranch: "fix-bug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedProj *project.Project
			var capturedCfg *config.RalphConfig
			var debugBranch string

			cmd := orchestrationWorkflow.NewWorkflowRunCmd(
				&mockWorWorkspaceSetupClient{
					setupFn: func(flags workspace.WorkspaceFlags) error {
						return nil
					},
				},
				&mockWorGitClient{
					fetchBranchFn: func(branch string) error { return nil },
					needsMergeFn:  func(branch string) (bool, error) { return false, nil },
				},
				&mockWorAIClient{},
				&mockWorRunnerClient{
					runLocalFn: func(proj *project.Project, cfg *config.RalphConfig) error {
						capturedProj = proj
						capturedCfg = cfg
						return nil
					},
				},
				&mockWorConfigClient{
					loadOptionalFn: func() (*config.RalphConfig, error) {
						return &config.RalphConfig{}, nil
					},
				},
				&mockWorProjectClient{
					loadFn: func(path string) (*project.Project, error) {
						return &project.Project{}, nil
					},
				},
				&mockWorDebugClient{
					setupFn: func(branch string) error {
						debugBranch = branch
						return nil
					},
				},
				&mockWorOutputClient{},
			)

			err := cmd.Run(tt.flags)
			require.NoError(t, err)

			if tt.debugBranch != "" {
				assert.Equal(t, tt.debugBranch, debugBranch)
			}

			if tt.wantBase != "" {
				assert.Equal(t, tt.wantBase, capturedProj.BaseBranch)
			}
			if tt.wantIter > 0 {
				assert.Equal(t, tt.wantIter, capturedProj.MaxIterations)
			}
			if tt.wantModel != "" {
				assert.Equal(t, tt.wantModel, capturedCfg.Model)
			}
		})
	}
}

func TestWorkflowRunCmd_WorkspaceSetupError(t *testing.T) {
	t.Parallel()

	cmd := orchestrationWorkflow.NewWorkflowRunCmd(
		&mockWorWorkspaceSetupClient{
			setupFn: func(flags workspace.WorkspaceFlags) error {
				return assert.AnError
			},
		},
		&mockWorGitClient{},
		&mockWorAIClient{},
		&mockWorRunnerClient{},
		&mockWorConfigClient{},
		&mockWorProjectClient{},
		&mockWorDebugClient{},
		&mockWorOutputClient{},
	)

	err := cmd.Run(orchestrationWorkflow.WorkflowRunFlags{ProjectPath: "test.yaml"})
	require.Error(t, err)
}

func TestWorkflowRunCmd_ProjectLoadError(t *testing.T) {
	t.Parallel()

	cmd := orchestrationWorkflow.NewWorkflowRunCmd(
		&mockWorWorkspaceSetupClient{},
		&mockWorGitClient{},
		&mockWorAIClient{},
		&mockWorRunnerClient{},
		&mockWorConfigClient{},
		&mockWorProjectClient{
			loadFn: func(path string) (*project.Project, error) {
				return nil, assert.AnError
			},
		},
		&mockWorDebugClient{},
		&mockWorOutputClient{},
	)

	err := cmd.Run(orchestrationWorkflow.WorkflowRunFlags{ProjectPath: "test.yaml"})
	require.Error(t, err)
}

func TestWorkflowRunCmd_RunnerCalledWithLoadedProjectAndConfig(t *testing.T) {
	t.Parallel()

	expectedProj := &project.Project{Slug: "test-project"}
	expectedCfg := &config.RalphConfig{Model: "test-model"}
	var capturedProj *project.Project
	var capturedCfg *config.RalphConfig

	cmd := orchestrationWorkflow.NewWorkflowRunCmd(
		&mockWorWorkspaceSetupClient{},
		&mockWorGitClient{
			fetchBranchFn: func(branch string) error { return nil },
			needsMergeFn:  func(branch string) (bool, error) { return false, nil },
		},
		&mockWorAIClient{},
		&mockWorRunnerClient{
			runLocalFn: func(proj *project.Project, cfg *config.RalphConfig) error {
				capturedProj = proj
				capturedCfg = cfg
				return nil
			},
		},
		&mockWorConfigClient{
			loadOptionalFn: func() (*config.RalphConfig, error) {
				return expectedCfg, nil
			},
		},
		&mockWorProjectClient{
			loadFn: func(path string) (*project.Project, error) {
				return expectedProj, nil
			},
		},
		&mockWorDebugClient{},
		&mockWorOutputClient{},
	)

	err := cmd.Run(orchestrationWorkflow.WorkflowRunFlags{
		ProjectPath:   "test.yaml",
		BaseBranch:    "main",
		ProjectBranch: "feature",
	})
	require.NoError(t, err)
	assert.Same(t, expectedProj, capturedProj)
	assert.Same(t, expectedCfg, capturedCfg)
}
