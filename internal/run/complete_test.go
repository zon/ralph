package run

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/testutil"
)

func TestFindCompleteProjects(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "complete-project.yaml",
			content: `name: complete-project
description: A complete project
requirements:
  - category: backend
    description: Feature 1
    items:
      - Item 1
      - Item 2
    passing: true
  - category: frontend
    description: Feature 2
    items:
      - Item 1
    passing: true
`,
			expected: true,
		},
		{
			name: "incomplete-project.yaml",
			content: `name: incomplete-project
description: An incomplete project
requirements:
  - category: backend
    description: Feature 1
    passing: true
  - category: frontend
    description: Feature 2
    passing: false
`,
			expected: false,
		},
		{
			name: "empty-requirements.yaml",
			content: `name: empty-project
description: Project with no requirements
requirements: []
`,
			expected: false,
		},
		{
			name:     "invalid-yaml.txt",
			content:  `not a valid yaml file`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.name)
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			require.NoError(t, err)

			projects, err := FindCompleteProjects(tmpDir)
			require.NoError(t, err)

			if tt.expected {
				absPath, _ := filepath.Abs(filePath)
				assert.Contains(t, projects, absPath)
			} else {
				absPath, _ := filepath.Abs(filePath)
				assert.NotContains(t, projects, absPath)
			}
		})
	}
}

func TestFindCompleteProjects_DirectoryDoesNotExist(t *testing.T) {
	_, err := FindCompleteProjects("/nonexistent/directory")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "directory does not exist")
}

func TestFindCompleteProjects_Recursive(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	completeFile := filepath.Join(subDir, "complete.yaml")
	completeContent := `name: complete
description: Complete project
requirements:
  - category: test
    description: Test
    passing: true
`
	err = os.WriteFile(completeFile, []byte(completeContent), 0644)
	require.NoError(t, err)

	incompleteFile := filepath.Join(tmpDir, "incomplete.yaml")
	incompleteContent := `name: incomplete
description: Incomplete project
requirements:
  - category: test
    description: Test
    passing: false
`
	err = os.WriteFile(incompleteFile, []byte(incompleteContent), 0644)
	require.NoError(t, err)

	projects, err := FindCompleteProjects(tmpDir)
	require.NoError(t, err)

	absComplete, _ := filepath.Abs(completeFile)
	absIncomplete, _ := filepath.Abs(incompleteFile)

	assert.Contains(t, projects, absComplete)
	assert.NotContains(t, projects, absIncomplete)
}

func TestRemoveAndCommit(t *testing.T) {
	workDir := setupCompleteTestRepo(t)
	t.Chdir(workDir)

	// Create project files
	files := []string{
		"projects/complete1.yaml",
		"projects/complete2.yaml",
	}
	for _, file := range files {
		err := os.MkdirAll(filepath.Dir(file), 0755)
		require.NoError(t, err)
		content := `name: test
description: Test project
requirements:
  - category: test
    description: Test requirement
    passing: true
`
		err = os.WriteFile(file, []byte(content), 0644)
		require.NoError(t, err)
	}

	ctx := testutil.NewContext()
	ctx.SetProjectFile("projects/complete1.yaml")

	absFiles := make([]string, len(files))
	for i, file := range files {
		absPath, _ := filepath.Abs(file)
		absFiles[i] = absPath
	}

	err := RemoveAndCommit(ctx, absFiles)
	assert.NoError(t, err)

	// Verify files are deleted
	for _, file := range files {
		_, err := os.Stat(file)
		assert.True(t, os.IsNotExist(err), "file %s should be deleted", file)
	}

	// Verify commit was created
	cmd := exec.Command("git", "log", "--oneline", "-1")
	cmd.Dir = workDir
	output, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(output), "chore: remove complete project files")
}

func TestRemoveAndCommit_EmptyList(t *testing.T) {
	ctx := testutil.NewContext()
	ctx.SetProjectFile("nonexistent.yaml")

	err := RemoveAndCommit(ctx, []string{})
	assert.NoError(t, err)
}

func setupCompleteTestRepo(t *testing.T) string {
	t.Helper()

	workDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Set user identity
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	// Create initial commit
	cmd = exec.Command("git", "commit", "--allow-empty", "-m", "initial commit")
	cmd.Dir = workDir
	require.NoError(t, cmd.Run())

	return workDir
}
