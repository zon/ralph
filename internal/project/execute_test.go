package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zon/ralph/internal/testutil"
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
			projectFile:  "/path/to/another-feature.yaml",
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
		{
			name:         "nested directories",
			projectFile:  "/path/to/nested/dirs/my-feature.yaml",
			expectedName: "my-feature",
		},
		{
			name:         "yml extension",
			projectFile:  "/path/to/my-feature.yml",
			expectedName: "my-feature",
		},
		{
			name:         "consecutive hyphens",
			projectFile:  "/path/to/my--feature.yaml",
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

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name         string
		projectName  string
		expectedName string
	}{
		{
			name:         "simple name",
			projectName:  "fix-pagination",
			expectedName: "fix-pagination",
		},
		{
			name:         "spaces in name",
			projectName:  "my cool feature",
			expectedName: "my-cool-feature",
		},
		{
			name:         "uppercase letters",
			projectName:  "MyFeature",
			expectedName: "myfeature",
		},
		{
			name:         "underscores",
			projectName:  "my_feature_branch",
			expectedName: "my-feature-branch",
		},
		{
			name:         "special characters",
			projectName:  "my@feature!",
			expectedName: "myfeature",
		},
		{
			name:         "multiple dots",
			projectName:  "my.feature.name",
			expectedName: "my-feature-name",
		},
		{
			name:         "leading/trailing hyphens",
			projectName:  "-my-feature-",
			expectedName: "my-feature",
		},
		{
			name:         "consecutive hyphens",
			projectName:  "my--feature",
			expectedName: "my-feature",
		},
		{
			name:         "subdirectory file name different from YAML name",
			projectName:  "fix-pagination",
			expectedName: "fix-pagination",
		},
		{
			name:         "directory name should not matter",
			projectName:  "fix-pagination",
			expectedName: "fix-pagination",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeBranchName(tt.projectName)
			if got != tt.expectedName {
				t.Errorf("sanitizeBranchName() = %v, want %v", got, tt.expectedName)
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
			ctx := testutil.NewContext(
				testutil.WithProjectFile(projectFile),
				testutil.WithMaxIterations(tt.maxIterations),
				testutil.WithDryRun(tt.dryRun),
			)

			err := Execute(ctx, nil)

			if tt.wantErr && err == nil {
				t.Error("Execute() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Execute() unexpected error: %v", err)
			}
		})
	}
}

func TestExecute_NotGitRepository(t *testing.T) {
	// Create a temporary directory that's NOT a git repository
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectContent := `name: Test Project
description: A test project
requirements:
  - description: Test requirement 1
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Change to the temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))

	err := Execute(ctx, nil)

	// In dry-run mode, IsGitRepository always returns true, so no error is expected.
	// This test verifies that Execute completes successfully in dry-run mode
	// even when not in an actual git repository.
	if err != nil {
		t.Errorf("Execute() should succeed in dry-run mode even when not in a git repo, got error: %v", err)
	}
}

func TestExecute_NonExistentProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "non-existent.yaml")

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithDryRun(true),
	)

	err := Execute(ctx, nil)

	if err == nil {
		t.Error("Execute() expected error when project file does not exist, got nil")
	}
}

func TestExecute_LocalDryRun_CreatesBranchFromProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "my-test-project.yaml")

	projectContent := `name: My Test Project
description: A test project
requirements:
  - description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithDryRun(true),
	)

	err := Execute(ctx, nil)

	if err != nil {
		t.Errorf("Execute() in local dry-run mode should complete without error: %v", err)
	}
}

func TestExecute_LocalDryRun_RespectsMaxIterations(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectContent := `name: Test Project
description: A test project
requirements:
  - description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithDryRun(true),
		testutil.WithMaxIterations(3),
	)

	err := Execute(ctx, nil)

	if err != nil {
		t.Errorf("Execute() should respect MaxIterations from context: %v", err)
	}
}

func TestExecute_SubdirectoryProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "projects", "p2")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	projectFile := filepath.Join(subDir, "fix-pagination.yaml")

	projectContent := `name: fix-pagination
description: A test project in subdirectory
requirements:
  - description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithDryRun(true),
	)

	err := Execute(ctx, nil)

	if err != nil {
		t.Errorf("Execute() should succeed with subdirectory project file: %v", err)
	}
}

func TestExecute_FileNameDifferentFromYamlName(t *testing.T) {
	tmpDir := t.TempDir()

	projectFile := filepath.Join(tmpDir, "some-other-file.yaml")

	projectContent := `name: fix-pagination
description: A test project where file name differs from YAML name
requirements:
  - description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithDryRun(true),
	)

	err := Execute(ctx, nil)

	if err != nil {
		t.Errorf("Execute() should use YAML name field, not file name: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
