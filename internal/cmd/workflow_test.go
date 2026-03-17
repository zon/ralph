package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/context"
)

func TestFetchBaseBranch_CreatesLocalTrackingBranch(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755)
	require.NoError(t, err)

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	err = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "branch", "main")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "checkout", "-b", "feature/test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	t.Chdir(tmpDir)

	remoteDir := t.TempDir()
	err = os.MkdirAll(filepath.Join(remoteDir, ".git"), 0755)
	require.NoError(t, err)

	cmd = exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "push", "-u", "origin", "feature/test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "branch", "-D", "main")
	cmd.Dir = tmpDir
	cmd.Run()

	localMainExists := func() bool {
		cmd := exec.Command("git", "rev-parse", "--verify", "main")
		cmd.Dir = tmpDir
		return cmd.Run() == nil
	}
	require.False(t, localMainExists(), "main should not exist locally initially")

	w := &WorkflowCmd{}
	ctx := &context.Context{}
	ctx.SetBaseBranch("main")

	err = w.fetchBaseBranch(ctx)
	require.NoError(t, err)

	require.True(t, localMainExists(), "main should exist locally after fetchBaseBranch")

	cmd = exec.Command("git", "log", "main..HEAD")
	cmd.Dir = tmpDir
	err = cmd.Run()
	assert.NoError(t, err, "git log main..HEAD should work with local main branch")
}

func TestFetchBaseBranch_FallsBackToPlainFetch(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755)
	require.NoError(t, err)

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	err = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "branch", "main")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "checkout", "-b", "feature/test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	t.Chdir(tmpDir)

	remoteDir := t.TempDir()
	err = os.MkdirAll(filepath.Join(remoteDir, ".git"), 0755)
	require.NoError(t, err)

	cmd = exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "push", "-u", "origin", "feature/test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "checkout", "main")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	w := &WorkflowCmd{}
	ctx := &context.Context{}
	ctx.SetBaseBranch("main")

	err = w.fetchBaseBranch(ctx)
	require.NoError(t, err, "Should fallback to plain fetch when already on the branch")
}
