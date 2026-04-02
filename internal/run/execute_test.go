package run

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/testutil"
)

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
			got := SanitizeBranchName(tt.projectName)
			assert.Equal(t, tt.expectedName, got, "SanitizeBranchName should return expected value")
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

func TestExecute_ValidProject(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project for requirement execution
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	require.NoError(t, os.WriteFile(projectFile, []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.NoError(t, err)
}

func TestExecute_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `name: Test
description: [invalid yaml structure
requirements:
  - not properly formatted
`

	require.NoError(t, os.WriteFile(projectFile, []byte(invalidYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.Error(t, err)
}

func TestExecute_EmptyRequirements(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "empty-reqs.yaml")

	emptyReqsYAML := `name: Test Project
description: Project with no requirements
requirements: []
`

	require.NoError(t, os.WriteFile(projectFile, []byte(emptyReqsYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.Error(t, err)
}

func TestExecute_BlockedMDExists(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	require.NoError(t, os.WriteFile(projectFile, []byte(projectYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "blocked.md"), []byte("Agent is blocked due to previous error"), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestExecute_BlockedMDContents(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	blockedContent := "This is the blocked reason from blocked.md"
	require.NoError(t, os.WriteFile(projectFile, []byte(projectYAML), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "blocked.md"), []byte(blockedContent), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), blockedContent)
}

func TestExecute_NormalizeTrailingNewlines(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAMLWithExcessNewlines := "name: Test Project\ndescription: Test project\nrequirements:\n  - id: req1\n    description: Test requirement\n    passing: false\n\n\n\n"

	require.NoError(t, os.WriteFile(projectFile, []byte(projectYAMLWithExcessNewlines), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	require.NoError(t, Execute(ctx, nil))

	content, err := os.ReadFile(projectFile)
	require.NoError(t, err)

	assert.True(t, strings.HasSuffix(string(content), "\n"))
	assert.False(t, strings.HasSuffix(string(content), "\n\n"))
}

func TestExecute_StartsServices(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	require.NoError(t, os.WriteFile(projectFile, []byte(projectYAML), 0644))

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.MkdirAll(ralphDir, 0755))

	configYAML := `services:
  - name: test-service
    command: echo
    args:
      - "test"
`
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configYAML), 0644))

	t.Chdir(tmpDir)

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithNoServices(false),
	)

	require.NoError(t, Execute(ctx, nil))
}

func TestExecute_WritesBlockedOnAgentFailure(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	t.Setenv("RALPH_MOCK_AI_FAIL", "true")
	t.Setenv("RALPH_WORKFLOW_EXECUTION", "true")
	logger.SetVerbose(true)

	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	require.NoError(t, os.WriteFile(projectFile, []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.Error(t, err)

	blockedContent, err := os.ReadFile(filepath.Join(tmpDir, "blocked.md"))
	require.NoError(t, err)

	assert.Contains(t, string(blockedContent), "Blocked")
	assert.Contains(t, string(blockedContent), "opencode execution failed")
}
