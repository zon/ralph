package project

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

func TestCommitChanges_UsesReportMd(t *testing.T) {
	// This test verifies that CommitChanges reads from report.md
	// We can't test actual git commits without a real git repo,
	// but we can test the report.md reading logic

	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create report.md with a commit message
	reportContent := "Add new feature\n\nImplemented feature X with tests"
	if err := os.WriteFile("report.md", []byte(reportContent), 0644); err != nil {
		t.Fatalf("Failed to create report.md: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile("project.yaml"))

	// In dry-run mode, this should read report.md (though not commit)
	err = CommitChanges(ctx, 1)
	if err != nil {
		t.Errorf("CommitChanges failed: %v", err)
	}

	// In dry-run mode, report.md should still exist
	if _, err := os.Stat("report.md"); err != nil {
		t.Errorf("report.md should exist in dry-run mode: %v", err)
	}
}

func TestCommitChanges_FallbackWhenNoReportMd(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Don't create report.md - should fall back to iteration-based message

	ctx := testutil.NewContext(testutil.WithProjectFile("project.yaml"))

	// Should use fallback message when report.md doesn't exist
	err = CommitChanges(ctx, 5)
	if err != nil {
		t.Errorf("CommitChanges failed: %v", err)
	}
}
