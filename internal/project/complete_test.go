package project

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
    passing: true
  - category: backend
    description: Feature 2
    items:
      - Item 2
    passing: true`,
			expected: true,
		},
		{
			name: "incomplete-project.yaml",
			content: `name: incomplete-project
description: An incomplete project
requirements:
  - category: backend
    description: Feature 1
    items:
      - Item 1
    passing: true
  - category: backend
    description: Feature 2
    items:
      - Item 2
    passing: false`,
			expected: false,
		},
		{
			name: "no-requirements-project.yaml",
			content: `name: no-requirements-project
description: A project with no requirements
requirements: []`,
			expected: false,
		},
		{
			name: "all-false-project.yaml",
			content: `name: all-false-project
description: A project with all requirements false
requirements:
  - category: backend
    description: Feature 1
    items:
      - Item 1
    passing: false
  - category: backend
    description: Feature 2
    items:
      - Item 2
    passing: false`,
			expected: false,
		},
		{
			name: "mixed-yaml-extension.yml",
			content: `name: mixed-extension-project
description: Project with .yml extension
requirements:
  - category: backend
    description: Feature 1
    items:
      - Item 1
    passing: true
  - category: backend
    description: Feature 2
    items:
      - Item 2
    passing: true`,
			expected: true,
		},
	}

	for _, tt := range tests {
		filePath := filepath.Join(tmpDir, tt.name)
		err := os.WriteFile(filePath, []byte(tt.content), 0644)
		require.NoError(t, err, "failed to write test file %s", tt.name)
	}

	completeProjects, err := FindCompleteProjects(tmpDir)
	require.NoError(t, err, "FindCompleteProjects should not error")

	expectedFiles := []string{
		filepath.Join(tmpDir, "complete-project.yaml"),
		filepath.Join(tmpDir, "mixed-yaml-extension.yml"),
	}

	assert.Len(t, completeProjects, len(expectedFiles), "FindCompleteProjects should return correct number of files")

	for _, expectedFile := range expectedFiles {
		assert.Contains(t, completeProjects, expectedFile, "FindCompleteProjects should contain expected file")
	}
}

func TestFindCompleteProjects_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	completeProjects, err := FindCompleteProjects(tmpDir)
	require.NoError(t, err, "FindCompleteProjects should not error")

	assert.Empty(t, completeProjects, "FindCompleteProjects should return empty list for empty directory")
}

func TestFindCompleteProjects_NonExistentDir(t *testing.T) {
	_, err := FindCompleteProjects("/non/existent/directory")
	assert.Error(t, err, "FindCompleteProjects should error for non-existent directory")
}

func TestFindCompleteProjects_RecursiveScanning(t *testing.T) {
	tmpDir := t.TempDir()

	subDir1 := filepath.Join(tmpDir, "sub1")
	subDir2 := filepath.Join(tmpDir, "sub2")
	subDir3 := filepath.Join(tmpDir, "sub1", "nested")
	err := os.MkdirAll(subDir1, 0755)
	require.NoError(t, err, "failed to create subdirectory")
	err = os.MkdirAll(subDir2, 0755)
	require.NoError(t, err, "failed to create subdirectory")
	err = os.MkdirAll(subDir3, 0755)
	require.NoError(t, err, "failed to create nested subdirectory")

	tests := []struct {
		name     string
		path     string
		content  string
		expected bool
	}{
		{
			name: "complete-project.yaml",
			path: tmpDir,
			content: `name: complete-project
description: A complete project at root
requirements:
  - category: backend
    description: Feature 1
    items:
      - Item 1
    passing: true`,
			expected: true,
		},
		{
			name:     "incomplete-project.yaml",
			path:     subDir1,
			content:  "name: incomplete\nrequirements:\n  - category: foo\n    passing: false",
			expected: false,
		},
		{
			name: "complete-in-subdir.yaml",
			path: subDir1,
			content: `name: complete-in-subdir
requirements:
  - category: backend
    description: Feature 1
    passing: true`,
			expected: true,
		},
		{
			name: "complete-in-another-subdir.yaml",
			path: subDir2,
			content: `name: complete-in-another-subdir
requirements:
  - category: backend
    description: Feature 1
    passing: true`,
			expected: true,
		},
		{
			name: "complete-in-nested.yaml",
			path: subDir3,
			content: `name: complete-in-nested
requirements:
  - category: backend
    description: Feature 1
    passing: true`,
			expected: true,
		},
	}

	for _, tt := range tests {
		filePath := filepath.Join(tt.path, tt.name)
		err := os.WriteFile(filePath, []byte(tt.content), 0644)
		require.NoError(t, err, "failed to write test file %s", tt.name)
	}

	completeProjects, err := FindCompleteProjects(tmpDir)
	require.NoError(t, err, "FindCompleteProjects should not error")

	expectedFiles := []string{
		filepath.Join(tmpDir, "complete-project.yaml"),
		filepath.Join(subDir1, "complete-in-subdir.yaml"),
		filepath.Join(subDir2, "complete-in-another-subdir.yaml"),
		filepath.Join(subDir3, "complete-in-nested.yaml"),
	}

	assert.Len(t, completeProjects, len(expectedFiles), "FindCompleteProjects should return correct number of files")

	for _, expectedFile := range expectedFiles {
		assert.Contains(t, completeProjects, expectedFile, "FindCompleteProjects should contain expected file")
	}
}

func TestFindCompleteProjects_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	invalidContent := `name: invalid-project
description: Invalid YAML
requirements:
  - category: backend
    description: Feature 1
    items:
      - Item 1
    passing: true
invalid yaml syntax here`

	filePath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(filePath, []byte(invalidContent), 0644)
	require.NoError(t, err, "failed to write invalid test file")

	completeProjects, err := FindCompleteProjects(tmpDir)
	require.NoError(t, err, "FindCompleteProjects should not error")

	assert.Empty(t, completeProjects, "FindCompleteProjects should return empty list for invalid YAML")
}

func TestRemoveAndCommit_EmptyFiles(t *testing.T) {
	ctx := testutil.NewContext()

	err := RemoveAndCommit(ctx, []string{})
	require.NoError(t, err, "RemoveAndCommit with empty files should not error")
}

func TestRemoveAndCommit_DryRun(t *testing.T) {
	// Set up a git repo for this test
	workDir := t.TempDir()
	t.Chdir(workDir)

	cmd := exec.Command("git", "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	remoteDir := t.TempDir()
	cmd = exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, out)
	}

	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, out)
	}

	testFile := "test-project.yaml"
	content := `name: test-project
description: Test project
requirements:
  - category: backend
    description: Feature 1
    items:
      - Item 1
    passing: true`

	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err, "failed to write test file")

	// Create an initial commit so we can commit the removal
	initFile := "init.txt"
	err = os.WriteFile(initFile, []byte("init"), 0644)
	require.NoError(t, err, "failed to write init file")
	cmd = exec.Command("git", "add", ".")
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Run()

	ctx := testutil.NewContext()

	err = RemoveAndCommit(ctx, []string{testFile})
	require.NoError(t, err, "RemoveAndCommit should not error")

	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err), "File should be removed")
}

func TestRemoveAndCommit_NonExistentFile(t *testing.T) {
	ctx := testutil.NewContext()

	err := RemoveAndCommit(ctx, []string{"/non/existent/file.yaml"})
	assert.Error(t, err, "RemoveAndCommit should return error for non-existent file")
}
