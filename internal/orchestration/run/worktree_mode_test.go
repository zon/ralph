package run

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

// ---------------------------------------------------------------------------
// Compile-time interface checks
// ---------------------------------------------------------------------------

func TestWorktreeClientInterfaceSatisfiedByRealClient(t *testing.T) {
	var _ WorktreeClient = (*git.Client)(nil)
	var _ LocalRunnerClient = (*Runner)(nil)
}

// ---------------------------------------------------------------------------
// Worktree client builders
// ---------------------------------------------------------------------------

func worktreeCheckedOutElsewhere() WorktreeClient {
	return &mockWorktreeClient{checkedOut: true}
}

func worktreeThatFailsList() WorktreeClient {
	return &mockWorktreeClient{
		BranchCheckedOutInWorktreeFunc: func(string, bool) (*git.WorktreeCommand, bool, error) {
			return nil, false, errors.New("failed to list worktrees")
		},
	}
}

func worktreeThatFailsCreate() WorktreeClient {
	return &mockWorktreeClient{
		CreateWorktreeFunc: func(string, bool) (*git.WorktreeCommand, error) {
			return nil, errors.New("failed to create worktree")
		},
	}
}

func worktreeThatFailsRemove() WorktreeClient {
	return &mockWorktreeClient{
		RemoveWorktreeFunc: func(string, bool) (*git.WorktreeCommand, error) {
			return nil, errors.New("failed to remove worktree")
		},
	}
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func worktreeBranchChecked(cmd *RunCmd) bool {
	if m, ok := cmd.worktree.(*mockWorktreeClient); ok {
		return m.BranchCheckedOutInWorktreeCalled
	}
	return false
}

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
// Scenario tests: worktree mode dispatch
// ---------------------------------------------------------------------------

func TestRunWorktreeCreatesWorktreeRunsLoopAndRemovesIt(t *testing.T) {
	ws := &mockWorkspaceClient{}
	wt := &mockWorktreeClient{}
	local := &mockLocalRunnerClient{}
	cmd := cmdWithMocks(
		cmdWithWorkspace(ws),
		cmdWithWorktree(wt),
		cmdWithLocal(local),
		cmdWithProject(projectWithSlug("my-project")),
	)

	err := cmd.Run(flagsWithMode(config.ModeWorktree))
	require.NoError(t, err)

	require.True(t, worktreeBranchChecked(cmd), "the branch is checked before the worktree is created")
	require.True(t, worktreeCreated(cmd), "a worktree is created for the project branch")
	require.Equal(t, "my-project", worktreeCreateBranch(cmd))
	require.True(t, local.RunLocalInWorktreeCalled, "the development loop runs in-process")
	require.False(t, local.RunLocalCalled, "the branch-switching loop is not used for worktree mode")
	require.True(t, worktreeRemoved(cmd), "the worktree is removed when the run ends")
	require.Equal(t, "my-project", wt.RemoveWorktreeBranch)
}

func TestRunWorktreeRunsLoopInsideWorktreeDirectory(t *testing.T) {
	var events []string
	ws := &mockWorkspaceClient{
		ChangeDirectoryFunc: func(dir string) error {
			if dir == "/sibling/repo-my-project" {
				events = append(events, "entered-worktree")
			}
			return nil
		},
	}
	local := &mockLocalRunnerClient{
		RunLocalInWorktreeFunc: func(*project.InputFile, *config.RalphConfig) error {
			events = append(events, "ran-loop")
			return nil
		},
	}
	cmd := cmdWithMocks(
		cmdWithWorkspace(ws),
		cmdWithLocal(local),
		cmdWithProject(projectWithSlug("my-project")),
	)

	err := cmd.Run(flagsWithMode(config.ModeWorktree))
	require.NoError(t, err)
	require.Equal(t, []string{"entered-worktree", "ran-loop"}, events, "the loop runs after changing into the worktree")
}

func TestRunWorktreeReturnsToMainCheckoutBeforeRemoval(t *testing.T) {
	ws := &mockWorkspaceClient{}
	cmd := cmdWithMocks(
		cmdWithWorkspace(ws),
		cmdWithProject(projectWithSlug("my-project")),
	)
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	require.NoError(t, cmd.Run(flagsWithMode(config.ModeWorktree)))

	require.Equal(t, []string{"", "/sibling/repo-my-project", originalDir}, ws.ChangedDirs,
		"the run changes back out of the worktree before removing it")
}

func TestRunWorktreeRemovesWorktreeWhenLoopFails(t *testing.T) {
	wt := &mockWorktreeClient{}
	local := &mockLocalRunnerClient{
		RunLocalInWorktreeFunc: func(*project.InputFile, *config.RalphConfig) error {
			return errors.New("iteration failed")
		},
	}
	cmd := cmdWithMocks(
		cmdWithWorktree(wt),
		cmdWithLocal(local),
		cmdWithProject(projectWithSlug("my-project")),
	)

	err := cmd.Run(flagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "iteration failed", "the loop's error is returned")
	require.True(t, worktreeRemoved(cmd), "the worktree is removed when the run fails")
}

func TestRunWorktreeRemovalErrorReportedWhenRunSucceeds(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithWorktree(worktreeThatFailsRemove()),
		cmdWithProject(projectWithSlug("my-project")),
	)

	err := cmd.Run(flagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to remove worktree")
}

func TestRunWorktreeLoopErrorTakesPrecedenceOverRemovalError(t *testing.T) {
	local := &mockLocalRunnerClient{
		RunLocalInWorktreeFunc: func(*project.InputFile, *config.RalphConfig) error {
			return errors.New("iteration failed")
		},
	}
	cmd := cmdWithMocks(
		cmdWithWorktree(worktreeThatFailsRemove()),
		cmdWithLocal(local),
		cmdWithProject(projectWithSlug("my-project")),
	)

	err := cmd.Run(flagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "iteration failed", "the loop's error is reported, not the removal error")
}

// ---------------------------------------------------------------------------
// Scenario tests: branch already checked out
// ---------------------------------------------------------------------------

func TestRunWorktreeBranchCheckedOutElsewhereReturnsErrorAndCreatesNothing(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithWorktree(worktreeCheckedOutElsewhere()),
		cmdWithProject(projectWithSlug("my-project")),
	)

	err := cmd.Run(flagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "branch 'my-project' is already checked out in another worktree")
	require.False(t, worktreeCreated(cmd), "no worktree is created when the branch is already checked out")
	require.False(t, worktreeRemoved(cmd))
	local := cmd.local.(*mockLocalRunnerClient)
	require.False(t, local.RunLocalInWorktreeCalled, "the loop does not run when the branch is already checked out")
}

// ---------------------------------------------------------------------------
// Scenario tests: worktree operation failures
// ---------------------------------------------------------------------------

func TestRunWorktreeListFailureAbortsBeforeCreate(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithWorktree(worktreeThatFailsList()),
		cmdWithProject(projectWithSlug("my-project")),
	)

	err := cmd.Run(flagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to list worktrees")
	require.False(t, worktreeCreated(cmd))
	require.False(t, worktreeRemoved(cmd))
}

func TestRunWorktreeCreateFailureAbortsWithoutRemoval(t *testing.T) {
	cmd := cmdWithMocks(
		cmdWithWorktree(worktreeThatFailsCreate()),
		cmdWithProject(projectWithSlug("my-project")),
	)

	err := cmd.Run(flagsWithMode(config.ModeWorktree))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create worktree")
	require.False(t, worktreeRemoved(cmd), "nothing is removed when no worktree was created")
}

func TestRunWorktreePassesResolvedConfigToLoop(t *testing.T) {
	local := &mockLocalRunnerClient{}
	cmd := cmdWithMocks(
		cmdWithConfig(configWithItems(".requirements")),
		cmdWithLocal(local),
		cmdWithProject(projectWithSlug("my-project")),
	)

	err := cmd.Run(flagsWithModeAndItems(config.ModeWorktree, ".spec.tasks"))
	require.NoError(t, err)
	require.True(t, local.RunLocalInWorktreeCalled)
	require.Equal(t, ".spec.tasks", local.LastConfig.Items)
}

// ---------------------------------------------------------------------------
// Scenario tests: input relocation into the worktree
// ---------------------------------------------------------------------------

// TestRelocateInput asserts the input is pointed at its copy in the worktree
// when it lives inside the starting repository, and returned unchanged when it
// is outside the repository or has no copy in the worktree.
func TestRelocateInput(t *testing.T) {
	repo := t.TempDir()
	worktreePath := filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"-branch")
	rel := filepath.Join("projects", "feature.yaml")
	require.NoError(t, os.MkdirAll(filepath.Join(worktreePath, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(worktreePath, rel), []byte("slug: feature\n"), 0644))

	cmd := cmdWithMocks()
	relocated := cmd.relocateInput(project.ForProjectInput(&project.Project{Slug: "feature", Path: filepath.Join(repo, rel)}), repo, worktreePath)
	require.Equal(t, filepath.Join(worktreePath, rel), relocated.Path())
	require.True(t, relocated.IsProject())

	missing := filepath.Join("projects", "untracked.yaml")
	unchanged := cmd.relocateInput(project.ForProjectInput(&project.Project{Slug: "untracked", Path: filepath.Join(repo, missing)}), repo, worktreePath)
	require.Equal(t, filepath.Join(repo, missing), unchanged.Path())

	outside := filepath.Join(t.TempDir(), "outside.yaml")
	unchanged = cmd.relocateInput(project.ForProjectInput(&project.Project{Slug: "outside", Path: outside}), repo, worktreePath)
	require.Equal(t, outside, unchanged.Path())

	unchanged = cmd.relocateInput(project.ForProjectInput(&project.Project{Slug: "no-root", Path: filepath.Join(repo, rel)}), "", worktreePath)
	require.Equal(t, filepath.Join(repo, rel), unchanged.Path())
}

// TestRunWorktreeRelocatesInputIntoWorktree asserts worktree mode points the
// input at its copy inside the worktree before the loop runs, so cleanup
// operates on the worktree checkout and the starting checkout stays untouched.
func TestRunWorktreeRelocatesInputIntoWorktree(t *testing.T) {
	repo := t.TempDir()
	testutil.InitGitRepo(t, repo)
	testutil.MakeInitialCommit(t, repo)
	t.Chdir(repo)

	rel := filepath.Join("projects", "retire-legacy.yaml")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, rel), []byte("slug: retire-legacy\nrequirements: []\n"), 0644))

	wtDir := filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"-retire-legacy")
	require.NoError(t, os.MkdirAll(filepath.Join(wtDir, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, rel), []byte("slug: retire-legacy\nrequirements: []\n"), 0644))

	inputPath := filepath.Join(repo, rel)
	local := &mockLocalRunnerClient{}
	wt := &mockWorktreeClient{
		CreateWorktreeFunc: func(string, bool) (*git.WorktreeCommand, error) {
			return &git.WorktreeCommand{Args: []string{"worktree", "add", "-b", "retire-legacy", wtDir}, Path: wtDir}, nil
		},
	}
	cmd := cmdWithMocks(
		cmdWithWorktree(wt),
		cmdWithLocal(local),
		cmdWithProject(&mockProjectRepo{
			InputFile: project.ForProjectInput(&project.Project{Slug: "retire-legacy", Path: inputPath}),
		}),
	)

	require.NoError(t, cmd.Run(flagsWithMode(config.ModeWorktree)))
	require.True(t, local.RunLocalInWorktreeCalled)
	require.Equal(t, filepath.Join(wtDir, rel), local.LastInput.Path())
}
