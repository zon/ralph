package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetHelpListsConfigOnly(t *testing.T) {
	repoRoot := findRepoRoot(t)
	binary := filepath.Join(t.TempDir(), "ralph")
	build := exec.Command("go", "build", "-o", binary, "./cmd/ralph")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(out))

	cmd := exec.Command(binary, "set", "--help")
	cmd.Dir = repoRoot
	out, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "config")
	assert.NotContains(t, string(out), "skills")
}
