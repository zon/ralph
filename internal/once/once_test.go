package once

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/context"
)

func TestExecute_DryRun(t *testing.T) {
	// Create a temporary test project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project for once command
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Execute in dry-run mode (should not fail)
	ctx := &context.Context{ProjectFile: projectFile, MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false} // dry-run, no verbose, no notify, no services
	err := Execute(ctx, nil)

	if err != nil {
		t.Errorf("Execute failed in dry-run mode: %v", err)
	}
}

func TestExecute_InvalidProjectFile(t *testing.T) {
	ctx := &context.Context{ProjectFile: "/nonexistent/project.yaml", MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}

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

	ctx := &context.Context{ProjectFile: projectFile, MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}
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

	ctx := &context.Context{ProjectFile: projectFile, MaxIterations: 10, DryRun: true, Verbose: false, NoNotify: true, NoServices: false}
	err := Execute(ctx, nil)

	if err == nil {
		t.Error("Expected error for project with no requirements, got nil")
	}
}
