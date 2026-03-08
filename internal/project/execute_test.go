package project

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

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
			assert.Equal(t, tt.expectedName, got, "extractBranchName should return expected value")
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
			assert.Equal(t, tt.expectedName, got, "sanitizeBranchName should return expected value")
		})
	}
}

func TestExecute_NonExistentProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "non-existent.yaml")

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))

	err := Execute(ctx, nil)

	assert.Error(t, err, "Execute should return error when project file does not exist")
}
