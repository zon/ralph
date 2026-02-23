package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/context"
)

func TestFindCompleteProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test project files
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

	// Write test files
	for _, tt := range tests {
		filePath := filepath.Join(tmpDir, tt.name)
		if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
			t.Fatalf("failed to write test file %s: %v", tt.name, err)
		}
	}

	// Test FindCompleteProjects
	completeProjects, err := FindCompleteProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindCompleteProjects() error = %v", err)
	}

	// Verify results
	expectedFiles := []string{
		filepath.Join(tmpDir, "complete-project.yaml"),
		filepath.Join(tmpDir, "mixed-yaml-extension.yml"),
	}

	if len(completeProjects) != len(expectedFiles) {
		t.Errorf("FindCompleteProjects() returned %d files, want %d", len(completeProjects), len(expectedFiles))
	}

	// Check that all expected files are present
	for _, expectedFile := range expectedFiles {
		found := false
		for _, actualFile := range completeProjects {
			if actualFile == expectedFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FindCompleteProjects() missing expected file: %s", expectedFile)
		}
	}

	// Check that no unexpected files are present
	for _, actualFile := range completeProjects {
		found := false
		for _, expectedFile := range expectedFiles {
			if actualFile == expectedFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FindCompleteProjects() returned unexpected file: %s", actualFile)
		}
	}
}

func TestFindCompleteProjects_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	completeProjects, err := FindCompleteProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindCompleteProjects() error = %v", err)
	}

	if len(completeProjects) != 0 {
		t.Errorf("FindCompleteProjects() returned %d files, want 0", len(completeProjects))
	}
}

func TestFindCompleteProjects_NonExistentDir(t *testing.T) {
	_, err := FindCompleteProjects("/non/existent/directory")
	if err == nil {
		t.Error("FindCompleteProjects() expected error for non-existent directory")
	}
}

func TestFindCompleteProjects_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid YAML file
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
	if err := os.WriteFile(filePath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write invalid test file: %v", err)
	}

	// Should skip invalid files and return empty list
	completeProjects, err := FindCompleteProjects(tmpDir)
	if err != nil {
		t.Fatalf("FindCompleteProjects() error = %v", err)
	}

	if len(completeProjects) != 0 {
		t.Errorf("FindCompleteProjects() returned %d files for invalid YAML, want 0", len(completeProjects))
	}
}

func TestRemoveAndCommit_EmptyFiles(t *testing.T) {
	ctx := &context.Context{
		DryRun:  true,
		Verbose: false,
	}

	err := RemoveAndCommit(ctx, []string{})
	if err != nil {
		t.Errorf("RemoveAndCommit() with empty files should return nil, got error: %v", err)
	}
}

func TestRemoveAndCommit_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test-project.yaml")
	content := `name: test-project
description: Test project
requirements:
  - category: backend
    description: Feature 1
    items:
      - Item 1
    passing: true`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ctx := &context.Context{
		DryRun:  true,
		Verbose: false,
	}

	err := RemoveAndCommit(ctx, []string{testFile})
	if err != nil {
		t.Errorf("RemoveAndCommit() in dry-run mode should not error, got: %v", err)
	}

	// File should still exist in dry-run mode
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("RemoveAndCommit() in dry-run mode should not delete files")
	}
}

func TestRemoveAndCommit_NonExistentFile(t *testing.T) {
	ctx := &context.Context{
		DryRun:  false,
		Verbose: false,
	}

	// Try to remove a non-existent file
	err := RemoveAndCommit(ctx, []string{"/non/existent/file.yaml"})
	if err == nil {
		t.Error("RemoveAndCommit() should return error for non-existent file")
	}
}
