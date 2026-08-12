package project

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/testutil"
)

// realCommitLog reads the commit messages of the current branch against a base
// branch using the real git wrapper, so completion can be read from an actual
// repository.
type realCommitLog struct{}

func (realCommitLog) CommitMessages(base string) ([]string, error) {
	return git.CommitMessages(base)
}

func TestCleanupScenario_CompletionStillReadableAfterCleanup(t *testing.T) {
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	testutil.MakeInitialCommit(t, dir)
	t.Chdir(dir)

	base, err := git.GetCurrentBranch()
	require.NoError(t, err)
	require.NoError(t, git.CheckoutOrCreateBranch("cleanup-branch"))

	require.NoError(t, os.MkdirAll("projects", 0o755))
	projectPath := "projects/test-project.yaml"
	require.NoError(t, os.WriteFile(projectPath, []byte("- one\n- two\n"), 0o644))
	require.NoError(t, git.StageFile(projectPath))
	require.NoError(t, git.Commit("chore: add project"))

	require.NoError(t, os.WriteFile("serializer.go", []byte("package main\n"), 0o644))
	require.NoError(t, git.StageFile("serializer.go"))
	require.NoError(t, git.Commit("feat: add serializer\n\nRalph item 0 completed\nRalph item 1 completed"))

	require.NoError(t, os.Remove(projectPath))
	require.NoError(t, git.StageFile(projectPath))
	require.NoError(t, git.CommitProjectRemoval(projectPath))

	client := NewClient(realCommitLog{}, &captureOutput{})
	indices, err := client.Complete(completedProject("one", "two"), base)
	require.NoError(t, err)
	require.Equal(t, []int{0, 1}, indices, "the trailers in the branch's history still report every item complete after cleanup deleted the project file")
}
