package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/context"
)

func TestExtractBranchName(t *testing.T) {
	tests := []struct {
		name         string
		projectFile  string
		expectedName string
	}{
		{
			name:         "simple yaml file",
			projectFile:  "/path/to/my-feature.yaml",
			expectedName: "my-feature",
		},
		{
			name:         "yml extension",
			projectFile:  "/path/to/another-feature.yml",
			expectedName: "another-feature",
		},
		{
			name:         "spaces in name",
			projectFile:  "/path/to/my cool feature.yaml",
			expectedName: "my-cool-feature",
		},
		{
			name:         "uppercase letters",
			projectFile:  "/path/to/MyFeature.yaml",
			expectedName: "myfeature",
		},
		{
			name:         "underscores",
			projectFile:  "/path/to/my_feature_branch.yaml",
			expectedName: "my-feature-branch",
		},
		{
			name:         "special characters",
			projectFile:  "/path/to/my@feature!.yaml",
			expectedName: "myfeature",
		},
		{
			name:         "multiple dots",
			projectFile:  "/path/to/my.feature.name.yaml",
			expectedName: "my-feature-name",
		},
		{
			name:         "leading/trailing hyphens",
			projectFile:  "/path/to/-my-feature-.yaml",
			expectedName: "my-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBranchName(tt.projectFile)
			if got != tt.expectedName {
				t.Errorf("extractBranchName() = %v, want %v", got, tt.expectedName)
			}
		})
	}
}

func TestExecute_DryRun(t *testing.T) {
	// Create a temporary project file for testing
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectContent := `name: Test Project
description: A test project
requirements:
  - description: Test requirement 1
    passing: false
  - description: Test requirement 2
    passing: true
`

	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	tests := []struct {
		name          string
		dryRun        bool
		maxIterations int
		wantErr       bool
	}{
		{
			name:          "dry-run mode",
			dryRun:        true,
			maxIterations: 5,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.NewContext(tt.dryRun, false, true, false)

			err := Execute(ctx, projectFile, tt.maxIterations, nil)

			if tt.wantErr && err == nil {
				t.Error("Execute() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Execute() unexpected error: %v", err)
			}
		})
	}
}

func TestExecute_InvalidProjectFile(t *testing.T) {
	ctx := context.NewContext(false, false, true, false)

	err := Execute(ctx, "/nonexistent/project.yaml", 10, nil)

	if err == nil {
		t.Error("Execute() with nonexistent file should return error")
	}
}
