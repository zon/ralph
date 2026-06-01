package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/testutil"
)

func TestRunGit(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantErr      bool
		checkTrimmed bool
	}{
		{
			name:    "successful command returns output",
			args:    []string{"rev-parse", "--show-toplevel"},
			wantErr: false,
		},
		{
			name:    "failed command returns output and error",
			args:    []string{"rev-parse", "nonexistent-ref"},
			wantErr: true,
		},
		{
			name:         "output is trimmed",
			args:         []string{"rev-parse", "--show-toplevel"},
			checkTrimmed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTestRepo(t)
			t.Chdir(dir)

			output, err := runGit(tt.args...)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.NotEmpty(t, output)
			if tt.checkTrimmed {
				assert.Equal(t, strings.TrimSpace(output), output)
			}
		})
	}
}

func TestDetectModifiedProjectFile(t *testing.T) {
	t.Run("returns empty when no project files exist", func(t *testing.T) {
		dir := t.TempDir()
		testutil.InitGitRepo(t, dir)
		t.Chdir(dir)

		result, err := DetectModifiedProjectFile("projects")
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("detects new project file", func(t *testing.T) {
		dir := t.TempDir()
		testutil.InitGitRepo(t, dir)
		t.Chdir(dir)

		require.NoError(t, os.MkdirAll("projects", 0755))
		projectFile := filepath.Join("projects", "fix-ai-error-handling.yaml")
		require.NoError(t, os.WriteFile(projectFile, []byte(`slug: fix-ai-error-handling
title: Fix AI error handling
requirements:
  - slug: fix-the-error
    description: Fix the error
    items:
      - Fix it
    passing: false
`), 0644))

		result, err := DetectModifiedProjectFile("projects")
		require.NoError(t, err)
		expected, err := filepath.Abs(projectFile)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("detects modified project file", func(t *testing.T) {
		dir := t.TempDir()
		testutil.InitGitRepo(t, dir)
		t.Chdir(dir)

		require.NoError(t, os.MkdirAll("projects", 0755))
		projectFile := filepath.Join("projects", "test-project.yaml")
		content := "name: test-project\ndescription: Test project\nrequirements: []\n"
		require.NoError(t, os.WriteFile(projectFile, []byte(content), 0644))

		for _, args := range [][]string{
			{"add", "projects/test-project.yaml"},
			{"commit", "-m", "Add project file"},
		} {
			c := exec.Command("git", args...)
			c.Dir = dir
			out, err := c.CombinedOutput()
			require.NoError(t, err, "git %v: %s", args, out)
		}

		require.NoError(t, os.WriteFile(projectFile, []byte(content+"\n# modified\n"), 0644))

		result, err := DetectModifiedProjectFile("projects")
		require.NoError(t, err)
		expected, err := filepath.Abs(projectFile)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("ignores non-yaml files", func(t *testing.T) {
		dir := t.TempDir()
		testutil.InitGitRepo(t, dir)
		t.Chdir(dir)

		require.NoError(t, os.MkdirAll("projects", 0755))
		require.NoError(t, os.WriteFile(filepath.Join("projects", "README.md"), []byte("# Projects\n"), 0644))

		result, err := DetectModifiedProjectFile("projects")
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testutil.InitGitRepo(t, dir)
	testutil.MakeInitialCommit(t, dir)
	return dir
}

func setupBareRemoteRepo(t *testing.T) (workDir, remoteDir string) {
	t.Helper()

	remoteDir = t.TempDir()
	c := exec.Command("git", "init", "--bare")
	c.Dir = remoteDir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git init --bare: %s", out)

	workDir = t.TempDir()
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

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# test\n"), 0644))
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "initial commit"},
		{"push", "origin", "HEAD"},
	} {
		c = exec.Command("git", args...)
		c.Dir = workDir
		out, err = c.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	return workDir, remoteDir
}
