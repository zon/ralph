package workspace

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
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
