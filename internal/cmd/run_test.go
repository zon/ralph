package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the Kong RunCmd struct are in internal/orchestration/run/cmd_test.go
// since the Run method now delegates to the orchestration layer.

func TestRunCmdFlagExtraIterations(t *testing.T) {
	repoRoot := findRepoRoot(t)
	binary := filepath.Join(t.TempDir(), "ralph")
	build := exec.Command("go", "build", "-o", binary, "./cmd/ralph")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(out))

	cmd := exec.Command(binary, "run", "--help")
	cmd.Dir = repoRoot
	out, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "--extra-iterations")
}

func TestRunCmdFlagItems(t *testing.T) {
	repoRoot := findRepoRoot(t)
	binary := filepath.Join(t.TempDir(), "ralph")
	build := exec.Command("go", "build", "-o", binary, "./cmd/ralph")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(out))

	cmd := exec.Command(binary, "run", "--help")
	cmd.Dir = repoRoot
	out, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "--items")
}

func TestRunCmdFlagCleanup(t *testing.T) {
	repoRoot := findRepoRoot(t)
	binary := filepath.Join(t.TempDir(), "ralph")
	build := exec.Command("go", "build", "-o", binary, "./cmd/ralph")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(out))

	cmd := exec.Command(binary, "run", "--help")
	cmd.Dir = repoRoot
	out, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "--cleanup")
}

func TestPassCommandRejected(t *testing.T) {
	repoRoot := findRepoRoot(t)
	binary := filepath.Join(t.TempDir(), "ralph")
	build := exec.Command("go", "build", "-o", binary, "./cmd/ralph")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(out))

	cmd := exec.Command(binary, "pass", "test.yaml", "test-slug")
	cmd.Dir = repoRoot
	out, err = cmd.CombinedOutput()
	require.Error(t, err, "ralph pass should exit non-zero")
	assert.Contains(t, string(out), "error")
}

func TestTopLevelHelpOmitsPass(t *testing.T) {
	repoRoot := findRepoRoot(t)
	binary := filepath.Join(t.TempDir(), "ralph")
	build := exec.Command("go", "build", "-o", binary, "./cmd/ralph")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(out))

	cmd := exec.Command(binary, "--help")
	cmd.Dir = repoRoot
	out, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.NotContains(t, string(out), "pass")
	assert.NotContains(t, string(out), "Mark a project requirement as passing or failing")
}

// findRepoRoot walks up from the working directory to find go.mod
func findRepoRoot(t *testing.T) string {
	t.Helper()
	// Start from the test's working directory
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
