package requirement

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/testutil"
)

func TestExecute_DryRun(t *testing.T) {
	// Create a temporary test project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project for requirement execution
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Execute in dry-run mode (should not fail)
	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile)) // dry-run, no verbose, no notify, no services
	err := Execute(ctx, nil)

	if err != nil {
		t.Errorf("Execute failed in dry-run mode: %v", err)
	}
}

func TestExecute_InvalidProjectFile(t *testing.T) {
	ctx := testutil.NewContext(testutil.WithProjectFile("/nonexistent/project.yaml"))

	// Test with non-existent file
	err := Execute(ctx, nil)
	if err == nil {
		t.Error("Expected error for non-existent project file, got nil")
	}
}

func TestExecute_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	invalidYAML := `name: Test
description: [invalid yaml structure
requirements:
  - not properly formatted
`

	if err := os.WriteFile(projectFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create invalid project file: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestExecute_EmptyRequirements(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "empty-reqs.yaml")

	// Project with no requirements should fail validation
	emptyReqsYAML := `name: Test Project
description: Project with no requirements
requirements: []
`

	if err := os.WriteFile(projectFile, []byte(emptyReqsYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	if err == nil {
		t.Error("Expected error for project with no requirements, got nil")
	}
}
