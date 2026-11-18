package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func InitGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		require.NoError(t, c.Run(), "git %v should succeed", args)
	}
}

func MakeInitialCommit(t *testing.T, dir string) {
	t.Helper()
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# test"), 0644))
	c := exec.Command("git", "add", "README.md")
	c.Dir = dir
	require.NoError(t, c.Run())
	c = exec.Command("git", "commit", "-m", "initial commit")
	c.Dir = dir
	require.NoError(t, c.Run())
}

// AddLocalRemote adds a bare origin remote and pushes main, so tests that call
// remote-aware git paths have an origin to query.
func AddLocalRemote(t *testing.T, dir string) string {
	t.Helper()
	remoteDir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--bare", remoteDir},
		{"remote", "add", "origin", remoteDir},
		{"push", "--set-upstream", "origin", "main"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		require.NoError(t, c.Run(), "git %v should succeed", args)
	}
	return remoteDir
}

func CreateRalphConfig(t *testing.T, dir string) {
	t.Helper()
	ralphDir := filepath.Join(dir, ".ralph")
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	configYAML := `defaultBranch: main
model: deepseek/deepseek-chat
`
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configYAML), 0644))
}
