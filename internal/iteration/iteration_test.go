package iteration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/testutil"
)

func TestRunIterationLoop_DryRun(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: test-project
description: Test project for iteration loop
requirements:
  - category: feature
    description: Add feature
    steps:
      - Step 1
      - Step 2
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))

	// Run iteration loop in dry-run mode
	iterations, err := RunIterationLoop(ctx, nil)
	if err != nil {
		t.Errorf("RunIterationLoop failed in dry-run: %v", err)
	}

	// In dry-run mode, it should return max iterations
	if iterations != 10 {
		t.Errorf("Expected 10 iterations, got %d", iterations)
	}
}

func TestCommitChanges_DryRun(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))

	// In dry-run mode, should not error
	err := CommitChanges(ctx, 1)
	if err != nil {
		t.Errorf("CommitChanges failed in dry-run: %v", err)
	}
}
