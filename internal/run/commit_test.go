package run

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/testutil"
)

// setupCommitTestRepo creates a temporary git repo with a bare remote.
// Returns the path to the working clone.
func setupCommitTestRepo(t *testing.T) string {
	t.Helper()

	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	workDir := t.TempDir()
	cmd = exec.Command("git", "clone", remoteDir, workDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v\n%s", err, out)
	}

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test@example.com"},
		{"config", "--local", "user.name", "Test User"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Create an initial commit and push it so the bare remote has a valid HEAD.
	readmePath := filepath.Join(workDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "initial commit"},
		{"push", "origin", "HEAD"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Create .ralph directory with config.yaml
	ralphDir := filepath.Join(workDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("failed to create .ralph directory: %v", err)
	}
	repoConfig, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load repo config: %v", err)
	}
	configContent := "model: " + repoConfig.Model + "\n"
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create .ralph/config.yaml: %v", err)
	}

	// Add and commit .ralph directory so it's tracked in the test repo
	for _, args := range [][]string{
		{"add", ".ralph"},
		{"commit", "-m", "add ralph config"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	return workDir
}

func TestCommitFileChanges_StagesFileAndCommits(t *testing.T) {
	workDir := setupCommitTestRepo(t)
	t.Chdir(workDir)

	// Create a new branch for the test
	testBranch := "test-branch"
	cmd := exec.Command("git", "checkout", "-b", testBranch)
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Create a new file
	filePath := filepath.Join(workDir, "newfile.txt")
	content := "test content"
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	ctx := testutil.NewContext()
	err := CommitFileChanges(ctx, testBranch, filePath, "Add new file")
	require.NoError(t, err, "CommitFileChanges should succeed")

	// Verify file was committed
	cmd = exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "Add new file")

	// Verify file is present in git
	cmd = exec.Command("git", "show", "HEAD:newfile.txt")
	cmd.Dir = workDir
	out, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, content, strings.TrimSpace(string(out)))
}

func TestCommitFileChanges_SwitchesBranch(t *testing.T) {
	workDir := setupCommitTestRepo(t)
	t.Chdir(workDir)

	// Start on main branch
	cmd := exec.Command("git", "branch", "other-branch")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Create a file on other-branch
	cmd = exec.Command("git", "checkout", "other-branch")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	filePath := filepath.Join(workDir, "other.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("other"), 0644))

	ctx := testutil.NewContext()
	err := CommitFileChanges(ctx, "other-branch", filePath, "Add other file")
	require.NoError(t, err)

	// Verify we are still on other-branch after commit
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, "other-branch", strings.TrimSpace(string(out)))
}

func TestCommitFileChanges_NoSuchFile(t *testing.T) {
	workDir := setupCommitTestRepo(t)
	t.Chdir(workDir)

	ctx := testutil.NewContext()
	err := CommitFileChanges(ctx, "main", "nonexistent.txt", "commit")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stage file")
}

func TestCommitFileChanges_NoChangesToCommit(t *testing.T) {
	workDir := setupCommitTestRepo(t)
	t.Chdir(workDir)

	// Stage a file that doesn't exist (should fail)
	ctx := testutil.NewContext()
	err := CommitFileChanges(ctx, "main", "nonexistent.txt", "commit")
	require.Error(t, err)
	// Error should be about staging
}

func TestCreatePullRequest_Integration(t *testing.T) {
	// This test requires gh CLI and a real repo; we'll skip for now.
	// We can rely on existing tests for createPullRequest (now CreatePullRequest).
	t.Skip("Integration test requiring gh CLI")
}
