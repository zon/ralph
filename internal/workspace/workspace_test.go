package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/output"
)

func TestChdir(t *testing.T) {
	tmpDir := t.TempDir()
	err := Chdir(tmpDir)
	require.NoError(t, err)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, tmpDir, cwd)
}

func TestChdirNonexistent(t *testing.T) {
	err := Chdir("/nonexistent/path/that/does/not/exist")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to change directory")
}

func TestConstants(t *testing.T) {
	require.Equal(t, "/secrets/opencode", DefaultOpenCodeSecretsDir)
	require.Equal(t, "/workspace", DefaultWorkspaceDir)
	require.Equal(t, "/workspace/repo", DefaultWorkDir)
}

func setupBareRemoteRepo(t *testing.T) (remoteDir string) {
	t.Helper()

	remoteDir = t.TempDir()
	c := exec.Command("git", "init", "--bare")
	c.Dir = remoteDir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git init --bare: %s", out)

	workDir := t.TempDir()
	c = exec.Command("git", "clone", remoteDir, workDir)
	out, err = c.CombinedOutput()
	require.NoError(t, err, "git clone: %s", out)

	for _, args := range [][]string{
		{"config", "--local", "user.email", "test@example.com"},
		{"config", "--local", "user.name", "Test User"},
	} {
		c = exec.Command("git", args...)
		c.Dir = workDir
		out, err = c.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# main branch\n"), 0644))
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "initial commit on main"},
		{"push", "origin", "HEAD"},
	} {
		c = exec.Command("git", args...)
		c.Dir = workDir
		out, err = c.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	for _, args := range [][]string{
		{"checkout", "-b", "feature/test"},
	} {
		c = exec.Command("git", args...)
		c.Dir = workDir
		out, err = c.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "feature.txt"), []byte("feature branch content\n"), 0644))
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "add feature.txt on feature/test"},
		{"push", "origin", "HEAD"},
	} {
		c = exec.Command("git", args...)
		c.Dir = workDir
		out, err = c.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	return remoteDir
}

func TestPrepareWorkspace(t *testing.T) {
	// Fix CWD in case an earlier test left it in a cleaned-up temp directory
	safeDir := t.TempDir()
	require.NoError(t, os.Chdir(safeDir))

	out := output.NewClient(os.Stdout, os.Stderr, false)

	t.Run("successfully checks out named branch", func(t *testing.T) {
		defer os.Chdir(safeDir)

		remoteDir := setupBareRemoteRepo(t)
		workDir := filepath.Join(t.TempDir(), "repo")

		err := PrepareWorkspace(out, remoteDir, "feature/test", workDir)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(workDir, "feature.txt"))
		assert.NoError(t, err, "feature.txt should exist from the feature/test branch")

		cwd, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, workDir, cwd)
	})

	t.Run("falls back to default branch when named branch does not exist", func(t *testing.T) {
		defer os.Chdir(safeDir)

		remoteDir := setupBareRemoteRepo(t)
		workDir := filepath.Join(t.TempDir(), "repo")

		err := PrepareWorkspace(out, remoteDir, "nonexistent-branch", workDir)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(workDir, "README.md"))
		assert.NoError(t, err, "README.md should exist from the default branch")

		cwd, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, workDir, cwd)
	})

	t.Run("returns error for invalid repo URL", func(t *testing.T) {
		defer os.Chdir(safeDir)

		workDir := filepath.Join(t.TempDir(), "repo")

		err := PrepareWorkspace(out, "/nonexistent/path/to/repo", "main", workDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clone repository")
	})
}
