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

func (realCommitLog) CurrentBranch() (string, error) {
	return git.GetCurrentBranch()
}

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
	require.NoError(t, git.Commit("feat: add serializer\n\ncleanup-branch-8xt8Szl\ncleanup-branch-4oxMK9s"))

	require.NoError(t, os.Remove(projectPath))
	require.NoError(t, git.StageFile(projectPath))
	require.NoError(t, git.CommitProjectRemoval(projectPath))

	client := NewClient(realCommitLog{}, &captureOutput{})
	hashes, err := client.Complete(completedProject("one", "two"), base)
	require.NoError(t, err)
	require.Equal(t, []string{"4oxMK9s", "8xt8Szl"}, hashes, "the trailers in the branch's history still report every item complete after cleanup deleted the project file")
}

func TestCleanupScenario_ArchitectureDocumentLeftUntouched(t *testing.T) {
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	testutil.MakeInitialCommit(t, dir)
	t.Chdir(dir)

	base, err := git.GetCurrentBranch()
	require.NoError(t, err)
	require.NoError(t, git.CheckoutOrCreateBranch("cleanup-branch"))

	require.NoError(t, os.MkdirAll("specs/features/ralph/export", 0o755))
	architecture := []byte("# Export Feature Architecture\n\nmodules: []\n")
	require.NoError(t, os.WriteFile("specs/features/ralph/export/architecture.yaml", architecture, 0o644))

	require.NoError(t, os.MkdirAll("projects", 0o755))
	projectPath := "projects/export.yaml"
	require.NoError(t, os.WriteFile(projectPath, []byte("slug: export\nfeature: specs/features/ralph/export\n"), 0o644))
	require.NoError(t, git.StageFile(projectPath))
	require.NoError(t, git.StageFile("specs/features/ralph/export/architecture.yaml"))
	require.NoError(t, git.Commit("chore: add project and feature architecture"))

	require.NoError(t, os.WriteFile("serializer.go", []byte("package main\n"), 0o644))
	require.NoError(t, git.StageFile("serializer.go"))
	require.NoError(t, git.Commit("feat: add exporter\n\ncleanup-branch-9izELVK"))

	client := NewClient(realCommitLog{}, &captureOutput{})
	proj, err := client.Resolve(projectPath, ".")
	require.NoError(t, err)
	complete, err := client.Complete(proj, base)
	require.NoError(t, err)
	require.Equal(t, []string{"9izELVK"}, complete, "the completed iteration is recorded from the branch's commit trailer")

	require.NoError(t, client.Remove(proj))
	require.NoError(t, git.CommitProjectRemoval(projectPath))

	after, err := os.ReadFile("specs/features/ralph/export/architecture.yaml")
	require.NoError(t, err)
	require.Equal(t, architecture, after, "the feature architecture document is left untouched by the completion flow")
}
