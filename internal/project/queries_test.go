package project

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/projectfile"
)

func writeResolveProjectFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestResolveBuildsProjectFromParsedDocument(t *testing.T) {
	path := writeResolveProjectFile(t, "csv-export.yaml", "slug: csv-export\ntitle: CSV Export\nrequirements:\n  - slug: one\n  - slug: two\n  - slug: three\n")

	proj, err := (&Client{}).Resolve(path, ".requirements")
	require.NoError(t, err)

	assert.Equal(t, "csv-export", proj.Slug)
	assert.Equal(t, "CSV Export", proj.Title)
	assert.Equal(t, path, proj.Path)
	require.Len(t, proj.Items, 3)
	assert.Equal(t, 0, proj.Items[0].Index)
	assert.Equal(t, "one", proj.Items[0].Key())
	assert.Equal(t, 2, proj.Items[2].Index)
	require.NotNil(t, proj.Doc)
	root, ok := proj.Doc.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "csv-export", root["slug"])
	assert.NotEmpty(t, proj.Doc.Raw)
}

func TestResolveUsesResolvedValuesVerbatim(t *testing.T) {
	path := writeResolveProjectFile(t, "tasks.yaml", "- Add a CSV serializer\n- slug: export-endpoint\n  description: Expose exports\n")

	proj, err := (&Client{}).Resolve(path, ".")
	require.NoError(t, err)

	require.Len(t, proj.Items, 2)
	assert.Equal(t, "Add a CSV serializer", proj.Items[0].Value)
	assert.Equal(t, "export-endpoint", proj.Items[1].Key())
}

func TestResolveJSONProject(t *testing.T) {
	path := writeResolveProjectFile(t, "issues.json", `{"slug": "issues", "tasks": ["one", "two"]}`)

	proj, err := (&Client{}).Resolve(path, ".tasks")
	require.NoError(t, err)

	require.Len(t, proj.Items, 2)
	assert.Equal(t, "issues", proj.Slug)
}

func TestResolveSlugFallsBackToBaseName(t *testing.T) {
	path := writeResolveProjectFile(t, "tasks.yaml", "- one\n- two\n")

	proj, err := (&Client{}).Resolve(path, ".")
	require.NoError(t, err)
	assert.Equal(t, "tasks", proj.Slug)
	assert.Equal(t, "tasks", proj.Title, "title falls back to the resolved slug")
}

func TestResolveReturnsErrorWhenFileDoesNotParse(t *testing.T) {
	path := writeResolveProjectFile(t, "broken.yaml", "slug: [unclosed\n")

	_, err := (&Client{}).Resolve(path, ".")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse project")
}

func TestResolveReturnsErrorWhenQueryYieldsNoItems(t *testing.T) {
	path := writeResolveProjectFile(t, "empty.yaml", "foo: 1\n")

	_, err := (&Client{}).Resolve(path, "empty")
	require.Error(t, err)
	assert.Equal(t, "item query yielded no items: empty", err.Error())
}

func TestResolveReturnsErrorWhenQueryYieldsOnlyEmptyItems(t *testing.T) {
	path := writeResolveProjectFile(t, "empty.yaml", "requirements:\n  -\n  - \"\"\n  - {}\n")

	_, err := (&Client{}).Resolve(path, ".requirements")
	require.Error(t, err)
	assert.Equal(t, "item query yielded no items: .requirements", err.Error())
}

func TestResolveIndexesItemsAfterDroppingEmptyEntries(t *testing.T) {
	path := writeResolveProjectFile(t, "tasks.yaml", "- slug: csv-serializer\n-\n- \"\"\n- slug: export-endpoint\n")

	proj, err := (&Client{}).Resolve(path, ".")
	require.NoError(t, err)

	require.Len(t, proj.Items, 2)
	assert.Equal(t, 0, proj.Items[0].Index)
	assert.Equal(t, "csv-serializer", proj.Items[0].Key())
	assert.Equal(t, 1, proj.Items[1].Index)
	assert.Equal(t, "export-endpoint", proj.Items[1].Key())
}

func TestResolveReturnsErrorWhenQueryFails(t *testing.T) {
	path := writeResolveProjectFile(t, "project.yaml", "foo: 1\n")

	_, err := (&Client{}).Resolve(path, ".foo.bar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "item query failed")
}

func TestIncompleteReturnsItemsNotCompleteInArrayOrder(t *testing.T) {
	c, _, _ := testClient(
		"feat: a\n\nRalph item 0 completed",
		"feat: b\n\nRalph item 2 (export-endpoint) completed",
	)
	proj := completedProject("a", "b", "c", "d")

	incomplete, err := c.Incomplete(proj, "main")
	require.NoError(t, err)
	require.Len(t, incomplete, 2)
	assert.Equal(t, 1, incomplete[0].Index)
	assert.Equal(t, 3, incomplete[1].Index)
}

func TestIncompleteEmptyWhenAllComplete(t *testing.T) {
	c, _, _ := testClient("Ralph item 0 completed\nRalph item 1 completed")
	proj := completedProject("a", "b")

	incomplete, err := c.Incomplete(proj, "main")
	require.NoError(t, err)
	assert.Empty(t, incomplete, "an empty result is the iteration loop's exit condition")
}

func TestIncompleteSurfacesCommitLogError(t *testing.T) {
	log := &stubCommitLog{err: errors.New("boom")}
	client := NewClient(log, &captureOutput{})

	_, err := client.Incomplete(completedProject("a"), "main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestIncompletePreservesKeyedValues(t *testing.T) {
	c, _, _ := testClient("Ralph item 1 (b) completed")
	proj := completedProject(map[string]any{"slug": "a"}, map[string]any{"slug": "b"})

	incomplete, err := c.Incomplete(proj, "main")
	require.NoError(t, err)
	require.Len(t, incomplete, 1)
	assert.Equal(t, 0, incomplete[0].Index)
	assert.Equal(t, "a", incomplete[0].Key())
}

func TestIncompleteErrorNamesIncompleteItems(t *testing.T) {
	c, _, _ := testClient("Ralph item 0 completed")
	proj := completedProject(
		map[string]any{"slug": "csv-serializer"},
		"plain string item",
		map[string]any{"slug": "another-item"},
	)

	err := c.IncompleteError(proj, "main")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExtraIterationsReached)
	msg := err.Error()
	assert.Contains(t, msg, "item 1")
	assert.Contains(t, msg, "item 2 (another-item)")
	assert.NotContains(t, msg, "requirements still failing")
}

func TestIncompleteErrorNilWhenAllComplete(t *testing.T) {
	c, _, _ := testClient("Ralph item 0 completed")
	proj := completedProject("a")

	err := c.IncompleteError(proj, "main")
	require.NoError(t, err)
}

func TestRemoveDeletesProjectFile(t *testing.T) {
	path := writeResolveProjectFile(t, "project.yaml", "- one\n")
	require.FileExists(t, path)

	err := projectfile.Remove(path)
	require.NoError(t, err)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "file should be deleted")
}

func TestRemoveMissingFileReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	err := projectfile.Remove(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove")
}

func TestClientRemoveDeletesAndStagesDeletion(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	t.Chdir(dir)

	path := "project.yaml"
	require.NoError(t, os.WriteFile(path, []byte("- one\n- two\n"), 0o644))
	runGit(t, "add", path)
	runGit(t, "commit", "-m", "feat: add project")

	c := &Client{}
	proj := &Project{Path: path}
	require.NoError(t, c.Remove(proj))

	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "file should be deleted")

	status := runGit(t, "status", "--porcelain")
	assert.Contains(t, status, "D  project.yaml", "the deletion should be staged")
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	} {
		runGitIn(t, dir, args...)
	}
}

func runGit(t *testing.T, args ...string) string {
	t.Helper()
	return runGitIn(t, ".", args...)
}

func runGitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, out)
	return string(out)
}
