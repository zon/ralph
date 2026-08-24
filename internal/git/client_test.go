package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/git"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/testutil"
)

func TestGitClientNew(t *testing.T) {
	ctx := context.NewContext()
	client := git.NewClient(ctx)
	require.NotNil(t, client)
	var _ orchestrationRun.GitClient = client
}

func TestGitClientBlockedFileExists(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	client := git.NewClient(context.NewContext())

	t.Run("returns false when no blocked.md exists", func(t *testing.T) {
		assert.False(t, client.BlockedFileExists())
	})

	t.Run("returns true when blocked.md exists in repo root", func(t *testing.T) {
		blockedPath := filepath.Join(workDir, "blocked.md")
		require.NoError(t, os.WriteFile(blockedPath, []byte("blocked"), 0644))
		assert.True(t, client.BlockedFileExists())
	})
}

func TestGitClientWriteBlockedFile(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	client := git.NewClient(context.NewContext())
	err := &testBlockedError{"connection refused"}

	client.WriteBlockedFile(err)

	blockedPath := filepath.Join(workDir, "blocked.md")
	data, readErr := os.ReadFile(blockedPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "connection refused")
	assert.Contains(t, string(data), "# Blocked")
}

func TestGitClientHasChanges(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	client := git.NewClient(context.NewContext())

	t.Run("returns false with clean working tree", func(t *testing.T) {
		assert.False(t, client.HasChanges())
	})

	t.Run("returns true after modifying a file", func(t *testing.T) {
		require.NoError(t, os.WriteFile("new.txt", []byte("content"), 0644))
		assert.True(t, client.HasChanges())
	})
}

func TestGitClientReportExists(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	client := git.NewClient(context.NewContext())

	t.Run("returns false when no report.md exists", func(t *testing.T) {
		assert.False(t, client.ReportExists())
	})

	t.Run("returns true when report.md exists", func(t *testing.T) {
		require.NoError(t, os.WriteFile("report.md", []byte("report content"), 0644))
		assert.True(t, client.ReportExists())
	})
}

func TestGitClientCommitFromReport(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	ctx := context.NewContext()
	client := git.NewClient(ctx)

	reportContent := "Implement requirement: adapter-git"
	require.NoError(t, os.WriteFile("report.md", []byte(reportContent), 0644))
	require.NoError(t, os.WriteFile("newfile.txt", []byte("change"), 0644))

	err := client.CommitFromReport("test-slug")
	require.NoError(t, err)

	_, err = os.Stat("report.md")
	assert.True(t, os.IsNotExist(err), "report.md should be deleted after commit")
}

func TestGitClientCommitFromReportPreservesTrailer(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	ctx := context.NewContext()
	client := git.NewClient(ctx)

	const trailer = "csv-export-0"
	reportContent := "feat: add serializer\n\n" + trailer + "\n"
	require.NoError(t, os.WriteFile("report.md", []byte(reportContent), 0644))
	require.NoError(t, os.WriteFile("newfile.txt", []byte("change"), 0644))

	err := client.CommitFromReport("test-slug")
	require.NoError(t, err)

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	msg := strings.TrimRight(string(out), "\n")
	assert.Equal(t, strings.TrimRight(reportContent, "\n"), msg, "the report is committed verbatim, trailer included")
	assert.Equal(t, trailer, msg[strings.LastIndex(msg, "\n")+1:], "the completion trailer survives as the last line, neither appended to nor removed")
}

func TestGitClientCommitFromReportFailsWhenNoReport(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	client := git.NewClient(context.NewContext())

	err := client.CommitFromReport("test-slug")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report.md")
}

func TestGitClientCommitIterationAndPush(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	ctx := context.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	client := git.NewClient(ctx)

	baseBranch := currentBranch(t, workDir)
	baseHead := revParse(t, workDir, "HEAD")

	require.NoError(t, client.SwitchToLoopBranch("fmt"), "the loop branch is switched to before the iteration runs")

	reportContent := "Implement requirement: adapter-git"
	require.NoError(t, os.WriteFile("report.md", []byte(reportContent), 0644))
	require.NoError(t, os.WriteFile("newfile.txt", []byte("change"), 0644))

	err := client.CommitIterationAndPush("fmt")
	require.NoError(t, err)

	assert.Equal(t, "loop-fmt", currentBranch(t, workDir), "the client switches to the loop branch")
	assert.Equal(t, revParse(t, workDir, "HEAD"), revParse(t, workDir, "origin/loop-fmt"), "the iteration commit is pushed to the loop branch")
	assert.Equal(t, baseHead, revParse(t, workDir, baseBranch), "the base branch still points at its original HEAD, so the loop branch was created from it")
	_, err = os.Stat("report.md")
	assert.True(t, os.IsNotExist(err), "report.md should be deleted after the iteration commit")
	assert.Equal(t, reportContent, lastCommitMessage(t, workDir), "the last commit message is the report content")
	assert.NotContains(t, lsTreeFiles(t, workDir, "origin/loop-fmt"), "report.md", "the pushed loop branch tree must not contain report.md")
	assert.Empty(t, gitStatusPorcelain(t, workDir), "the working tree must be clean after the iteration commit")
}

func TestGitClientCommitIterationAndPushReusesExistingBranch(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	ctx := context.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	client := git.NewClient(ctx)

	require.NoError(t, client.SwitchToLoopBranch("fmt"), "the loop branch is switched to before the iterations run")

	firstReport := "iteration one"
	require.NoError(t, os.WriteFile("report.md", []byte(firstReport), 0644))
	require.NoError(t, os.WriteFile("file1.txt", []byte("one"), 0644))
	require.NoError(t, client.CommitIterationAndPush("fmt"))

	secondReport := "iteration two"
	require.NoError(t, os.WriteFile("report.md", []byte(secondReport), 0644))
	require.NoError(t, os.WriteFile("file2.txt", []byte("two"), 0644))

	err := client.CommitIterationAndPush("fmt")
	require.NoError(t, err)

	assert.Equal(t, "loop-fmt", currentBranch(t, workDir), "the client stays on the loop branch")
	assert.Equal(t, revParse(t, workDir, "HEAD"), revParse(t, workDir, "origin/loop-fmt"), "the second iteration commit is pushed to the loop branch")
	assert.Equal(t, secondReport, lastCommitMessage(t, workDir), "the last commit message is the second report content")
	assert.NotContains(t, lsTreeFiles(t, workDir, "origin/loop-fmt"), "report.md", "the pushed loop branch tree must not contain report.md")
	assert.Empty(t, gitStatusPorcelain(t, workDir), "the working tree must be clean after the iteration commit")
}

func TestGitClientCommitIterationAndPushFailsWhenNoReport(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)

	client := git.NewClient(context.NewContext())

	err := client.CommitIterationAndPush("fmt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report.md")
}

func TestGitClientSwitchToLoopBranch(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	ctx := context.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	client := git.NewClient(ctx)

	require.NoError(t, client.SwitchToLoopBranch("fmt"))
	assert.Equal(t, "loop-fmt", currentBranch(t, workDir), "the client switches to the loop branch, creating it from the current branch")
	assert.Empty(t, gitStatusPorcelain(t, workDir), "the working tree stays clean during the switch")
}

// TestGitClientSwitchToLoopBranchReusesDivergedRemoteBranch reproduces the
// failure that stashing could not fix: the loop branch from a prior run has
// diverged from the base branch and the agent's edits differ from it. Because
// the switch happens before the agent runs, the divergent loop branch checks
// out cleanly, the agent edits on the loop branch itself, and the iteration
// commits on top of the prior work.
func TestGitClientSwitchToLoopBranchReusesDivergedRemoteBranch(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	require.NoError(t, os.WriteFile("note.txt", []byte("hello world"), 0644))
	require.NoError(t, git.StageFile("note.txt"))
	require.NoError(t, git.Commit("chore: add note.txt"))
	_, err := git.Push(nil, "main")
	require.NoError(t, err)

	require.NoError(t, git.CreateBranch("loop-fmt"))
	require.NoError(t, os.WriteFile("note.txt", []byte("diverged state"), 0644))
	require.NoError(t, git.StageFile("note.txt"))
	require.NoError(t, git.Commit("chore: prior run edits"))
	_, err = git.Push(nil, "loop-fmt")
	require.NoError(t, err)
	require.NoError(t, git.CheckoutBranch("main"))

	ctx := context.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	client := git.NewClient(ctx)

	require.NoError(t, client.SwitchToLoopBranch("fmt"), "the divergent loop branch switches in cleanly")
	assert.Equal(t, "loop-fmt", currentBranch(t, workDir))
	content, err := os.ReadFile("note.txt")
	require.NoError(t, err)
	assert.Equal(t, "diverged state", string(content), "the loop branch's own state is checked out")

	require.NoError(t, os.WriteFile("report.md", []byte("iteration report"), 0644))
	require.NoError(t, os.WriteFile("note.txt", []byte("diverged state with agent edit"), 0644))
	require.NoError(t, client.CommitIterationAndPush("fmt"))

	assert.Equal(t, revParse(t, workDir, "HEAD"), revParse(t, workDir, "origin/loop-fmt"), "the iteration commit is pushed to the loop branch")
	assert.Equal(t, "iteration report", lastCommitMessage(t, workDir), "the last commit message is the report content")
	assert.Empty(t, gitStatusPorcelain(t, workDir), "the working tree must be clean after the iteration commit")
}

func TestGitClientCommitGeneratedArtifacts(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	client := git.NewClient(context.NewContext())

	require.NoError(t, os.MkdirAll(filepath.Join(workDir, "projects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "projects", "test.yaml"), []byte("slug: my-feature\n"), 0644))

	err := client.CommitGeneratedArtifacts("my-feature")
	require.NoError(t, err)

	cmd := exec.Command("git", "log", "-1", "--format=%B")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, "chore: generate project for my-feature", strings.TrimSpace(string(out)))
}

func TestGitClientCommitGeneratedArtifactsNoChanges(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	client := git.NewClient(context.NewContext())

	err := client.CommitGeneratedArtifacts("empty-feature")
	require.Error(t, err)
}

func TestGitClientCommitProjectRemovalPushesToRemote(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	projectPath := "projects/test-project.yaml"
	require.NoError(t, os.MkdirAll(filepath.Dir(projectPath), 0755))
	require.NoError(t, os.WriteFile(projectPath, []byte("slug: test-project\n"), 0644))
	require.NoError(t, git.StageFile(projectPath))
	require.NoError(t, git.Commit("chore: add project"))
	_, err := git.Push(nil, "main")
	require.NoError(t, err)

	require.NoError(t, os.Remove(projectPath))
	require.NoError(t, git.StageFile(projectPath))

	client := git.NewClient(context.NewContext())
	require.NoError(t, client.CommitProjectRemoval(projectPath))

	assert.Equal(t, "chore: clean up completed project projects/test-project.yaml", lastCommitMessage(t, workDir))
	assert.Equal(t, revParse(t, workDir, "HEAD"), revParse(t, workDir, "origin/main"), "the cleanup commit is pushed to the remote")
}

func TestGitClientCommitOrchestrationRemovalPushesToRemote(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	testutil.InitGitRepo(t, workDir)
	testutil.MakeInitialCommit(t, workDir)
	setupLocalRemote(t, workDir)

	orchPath := "specs/features/ralph/export/orchestration.md"
	require.NoError(t, os.MkdirAll(filepath.Dir(orchPath), 0755))
	require.NoError(t, os.WriteFile(orchPath, []byte("# orchestration\n"), 0644))
	require.NoError(t, git.StageFile(orchPath))
	require.NoError(t, git.Commit("chore: add orchestration"))
	_, err := git.Push(nil, "main")
	require.NoError(t, err)

	require.NoError(t, os.Remove(orchPath))
	require.NoError(t, git.StageFile(orchPath))

	client := git.NewClient(context.NewContext())
	require.NoError(t, client.CommitOrchestrationRemoval("export"))

	assert.Equal(t, "chore: remove orchestration doc before PR", lastCommitMessage(t, workDir))
	assert.Equal(t, revParse(t, workDir, "HEAD"), revParse(t, workDir, "origin/main"), "the orchestration removal commit is pushed to the remote")
}

func revParse(t *testing.T, dir, ref string) string {
	t.Helper()
	c := exec.Command("git", "rev-parse", ref)
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git rev-parse %s", ref)
	return strings.TrimSpace(string(out))
}

func currentBranch(t *testing.T, dir string) string {
	t.Helper()
	c := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git rev-parse --abbrev-ref HEAD failed")
	return strings.TrimSpace(string(out))
}

func lastCommitMessage(t *testing.T, dir string) string {
	t.Helper()
	c := exec.Command("git", "log", "-1", "--format=%B")
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git log failed")
	return strings.TrimSpace(string(out))
}

func setupLocalRemote(t *testing.T, dir string) {
	t.Helper()
	bareDir := t.TempDir()

	c := exec.Command("git", "init", "--bare", bareDir)
	c.Dir = dir
	require.NoError(t, c.Run())

	c = exec.Command("git", "remote", "add", "origin", bareDir)
	c.Dir = dir
	require.NoError(t, c.Run())

	c = exec.Command("git", "push", "--set-upstream", "origin", "main")
	c.Dir = dir
	require.NoError(t, c.Run())
}

func lsTreeFiles(t *testing.T, dir, ref string) string {
	t.Helper()
	c := exec.Command("git", "ls-tree", "-r", ref, "--name-only")
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git ls-tree -r %s --name-only", ref)
	return string(out)
}

func gitStatusPorcelain(t *testing.T, dir string) string {
	t.Helper()
	c := exec.Command("git", "status", "--porcelain")
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git status --porcelain failed")
	return string(out)
}

type testBlockedError struct {
	msg string
}

func (e *testBlockedError) Error() string {
	return e.msg
}
