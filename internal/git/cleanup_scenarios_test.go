package git

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/trailer"
)

func TestCleanupScenario_CommitCarriesNoTrailer(t *testing.T) {
	dir := setupTestRepo(t)
	t.Chdir(dir)

	base, err := GetCurrentBranch()
	require.NoError(t, err)
	require.NoError(t, CheckoutOrCreateBranch("cleanup-branch"))

	require.NoError(t, os.MkdirAll("projects", 0o755))
	projectPath := "projects/test-project.yaml"
	require.NoError(t, os.WriteFile(projectPath, []byte("slug: test-project\n"), 0o644))
	require.NoError(t, StageFile(projectPath))
	require.NoError(t, Commit("chore: add project"))

	require.NoError(t, os.WriteFile("serializer.go", []byte("package main\n"), 0o644))
	require.NoError(t, StageFile("serializer.go"))
	require.NoError(t, Commit("feat: add serializer\n\ncleanup-branch-0"))

	before, err := CommitMessages(base)
	require.NoError(t, err)
	beforeRefs := trailer.Parse(strings.Join(before, "\n"))
	require.NotEmpty(t, beforeRefs, "the branch history carries a completion trailer before the cleanup commit")

	require.NoError(t, os.Remove(projectPath))
	require.NoError(t, StageFile(projectPath))
	require.NoError(t, CommitProjectRemoval(projectPath))

	after, err := CommitMessages(base)
	require.NoError(t, err)
	require.NotEmpty(t, after)

	cleanupMessage := after[0]
	assert.Equal(t, "chore: clean up completed project projects/test-project.yaml", strings.TrimSpace(cleanupMessage))
	assert.Empty(t, trailer.Parse(cleanupMessage), "no completion trailer is parsed from the cleanup commit")

	afterRefs := trailer.Parse(strings.Join(after, "\n"))
	require.Len(t, afterRefs, len(beforeRefs), "the completion record in the branch history is unchanged")
	for i := range beforeRefs {
		assert.Equal(t, beforeRefs[i], afterRefs[i], "the completion record already in the branch's history is unchanged")
	}
}

func TestCleanupItem_CommitContainsOnlyProjectFileDeletion(t *testing.T) {
	dir := setupTestRepo(t)
	t.Chdir(dir)

	require.NoError(t, os.MkdirAll("projects", 0o755))
	projectPath := "projects/test-project.yaml"
	require.NoError(t, os.WriteFile(projectPath, []byte("slug: test-project\n"), 0o644))
	require.NoError(t, os.WriteFile("README.md", []byte("# project\n"), 0o644))
	require.NoError(t, StageFile(projectPath))
	require.NoError(t, StageFile("README.md"))
	require.NoError(t, Commit("chore: add project and readme"))

	require.NoError(t, os.Remove(projectPath))
	require.NoError(t, StageFile(projectPath))
	require.NoError(t, os.WriteFile("README.md", []byte("# project\n\nchanged\n"), 0o644))

	require.NoError(t, CommitProjectRemoval(projectPath))

	out, err := runGit("show", "--name-status", "--format=", "HEAD")
	require.NoError(t, err)
	assert.Equal(t, "D\tprojects/test-project.yaml", strings.TrimSpace(out), "the cleanup commit contains no file changes other than the project file deletion")
	assert.True(t, IsFileModifiedOrNew("README.md"), "changes to other files are left uncommitted")
}
