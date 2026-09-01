package get

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
	"github.com/zon/ralph/internal/testutil"
)

func setupGetRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	testutil.CreateRalphConfig(t, dir)
	testutil.MakeInitialCommit(t, dir)
	for _, args := range [][]string{
		{"add", ".ralph"},
		{"commit", "-m", "add ralph config"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		require.NoError(t, c.Run(), "git %v should succeed", args)
	}
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, out)
}

// addTrailerCommit writes work.txt and commits it with the given message, so a
// completion trailer can be placed in the branch history.
func addTrailerCommit(t *testing.T, dir, message string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "work.txt"), []byte(message), 0644))
	runGit(t, dir, "add", "work.txt")
	runGit(t, dir, "commit", "-m", message)
}

func newGetCmd(buf *bytes.Buffer) *Cmd {
	ctx := testutil.NewContext()
	client := project.NewClient(git.NewClient(ctx), output.NewClient(io.Discard, io.Discard, false))
	return NewCmd(client, buf)
}

// itemHash returns the completion hash of a plain-string item, so a trailer
// commit can name the exact item a resolved array would produce.
func itemHash(v string) string {
	return project.NewItems([]any{v})[0].Hash()
}

func TestScenarioRepositoryLeftUntouched(t *testing.T) {
	dir := setupGetRepo(t)
	t.Chdir(dir)

	projectContent := "- item 0\n- item 1\n- item 2\n"
	projectPath := filepath.Join("projects", "csv-export.yaml")
	require.NoError(t, os.MkdirAll("projects", 0755))
	require.NoError(t, os.WriteFile(projectPath, []byte(projectContent), 0644))
	runGit(t, dir, "add", projectPath)
	runGit(t, dir, "commit", "-m", "add project file")
	runGit(t, dir, "checkout", "-b", "feature")
	addTrailerCommit(t, dir, "feat: serializer\n\nfeature-"+itemHash("item 0"))

	before, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	branchBefore, err := git.GetCurrentBranch()
	require.NoError(t, err)

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	var buf bytes.Buffer
	cmd := newGetCmd(&buf)

	require.NoError(t, cmd.Complete(cfg, Flags{}))
	assert.Equal(t, `["`+itemHash("item 0")+`"]`, strings.TrimSpace(buf.String()), "completion read from the log with no item array resolved")

	buf.Reset()
	require.NoError(t, cmd.Complete(cfg, Flags{ProjectFile: projectPath}))
	assert.Equal(t, `["`+itemHash("item 0")+`"]`, strings.TrimSpace(buf.String()), "resolved item array bounds the reported hashes")

	buf.Reset()
	require.NoError(t, cmd.Incomplete(cfg, Flags{ProjectFile: projectPath}, true))
	assert.Equal(t, "[1,2]", strings.TrimSpace(buf.String()), "the remaining items are the ones without a trailer")

	assert.False(t, git.HasUncommittedChanges(), "the working tree must remain clean")
	branchAfter, err := git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, branchBefore, branchAfter, "the current branch must be unchanged")
	after, err := os.ReadFile(projectPath)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the project file must remain byte-identical")
}

func TestScenarioCompletionScopedToCurrentBranch(t *testing.T) {
	dir := setupGetRepo(t)
	t.Chdir(dir)

	runGit(t, dir, "checkout", "-b", "develop")
	addTrailerCommit(t, dir, "feat: on develop\n\ndevelop-"+itemHash("item 1"))

	runGit(t, dir, "checkout", "-b", "feature")
	addTrailerCommit(t, dir, "feat: on feature\n\nfeature-"+itemHash("item 0"))

	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	require.Equal(t, "main", cfg.DefaultBranch)

	var buf bytes.Buffer
	cmd := newGetCmd(&buf)

	// Feature is forked from develop, so the develop-1 trailer is in the
	// main..HEAD log range. It names develop, not feature, so it is not counted.
	require.NoError(t, cmd.Complete(cfg, Flags{}))
	assert.Equal(t, `["`+itemHash("item 0")+`"]`, strings.TrimSpace(buf.String()), "completion on the current branch only")
}

func TestScenarioBaseOverridesConfiguredDefaultBranch(t *testing.T) {
	dir := setupGetRepo(t)
	t.Chdir(dir)

	runGit(t, dir, "checkout", "-b", "feature")
	addTrailerCommit(t, dir, "feat: on feature\n\nfeature-"+itemHash("item 0"))

	runGit(t, dir, "checkout", "-b", "develop")
	addTrailerCommit(t, dir, "feat: on develop\n\ndevelop-"+itemHash("item 2"))

	runGit(t, dir, "checkout", "feature")
	addTrailerCommit(t, dir, "feat: on feature\n\nfeature-"+itemHash("item 1"))

	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	require.Equal(t, "main", cfg.DefaultBranch)

	var buf bytes.Buffer
	cmd := newGetCmd(&buf)

	require.NoError(t, cmd.Complete(cfg, Flags{}))
	assert.Equal(t, `["`+itemHash("item 1")+`","`+itemHash("item 0")+`"]`, strings.TrimSpace(buf.String()), "the configured default branch bounds the log")

	buf.Reset()
	require.NoError(t, cmd.Complete(cfg, Flags{Base: "develop"}))
	assert.Equal(t, `["`+itemHash("item 1")+`"]`, strings.TrimSpace(buf.String()), "--base overrides the configured default branch")
}

func TestScenarioUnmatchedHashWarnedByGetComplete(t *testing.T) {
	dir := setupGetRepo(t)
	t.Chdir(dir)

	projectContent := "- item 0\n- item 1\n- item 2\n"
	projectPath := filepath.Join("projects", "csv-export.yaml")
	require.NoError(t, os.MkdirAll("projects", 0755))
	require.NoError(t, os.WriteFile(projectPath, []byte(projectContent), 0644))
	runGit(t, dir, "add", projectPath)
	runGit(t, dir, "commit", "-m", "add project file")

	runGit(t, dir, "checkout", "-b", "csv-export")
	addTrailerCommit(t, dir, "feat: unmatched\n\ncsv-export-"+itemHash("item 9"))

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	var warnBuf bytes.Buffer
	var buf bytes.Buffer
	ctx := testutil.NewContext()
	client := project.NewClient(git.NewClient(ctx), output.NewClient(&warnBuf, io.Discard, false))
	cmd := NewCmd(client, &buf)

	require.NoError(t, cmd.Complete(cfg, Flags{ProjectFile: projectPath}))
	assert.Equal(t, "[]", strings.TrimSpace(buf.String()), "a hash matching no resolved item is not reported as complete")
	assert.Contains(t, warnBuf.String(), itemHash("item 9"), "the warning names the unmatched hash")
	assert.Contains(t, warnBuf.String(), "matches no resolved item", "the warning says the hash matches no resolved item")
}
