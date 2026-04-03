package run

import (
	"os"
	"os/exec"
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

func TestExecute_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "invalid.yaml")
	invalidYAML := "name: Test\ndescription: [invalid yaml structure\nrequirements:\n  - not properly formatted\n"
	require.NoError(t, os.WriteFile(projectFile, []byte(invalidYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.Error(t, err)
}

func TestExecute_EmptyRequirements(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "empty-reqs.yaml")
	emptyReqsYAML := "name: Test Project\ndescription: Project with no requirements\nrequirements: []\n"
	require.NoError(t, os.WriteFile(projectFile, []byte(emptyReqsYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.Error(t, err)
}

// projectYAML is a minimal valid project used across development iteration tests.
const projectYAML = `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

func TestExecute_ValidProject(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := ExecuteDevelopmentIteration(ctx, nil)

	require.NoError(t, err)
}

func TestExecute_BlockedMDExists(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))
	require.NoError(t, os.WriteFile("blocked.md", []byte("Agent is blocked due to previous error"), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := ExecuteDevelopmentIteration(ctx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestExecute_BlockedMDContents(t *testing.T) {
	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))
	blockedContent := "This is the blocked reason from blocked.md"
	require.NoError(t, os.WriteFile("blocked.md", []byte(blockedContent), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := ExecuteDevelopmentIteration(ctx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), blockedContent)
}

func TestExecute_NoBlockedMD(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := ExecuteDevelopmentIteration(ctx, nil)

	require.NoError(t, err)
}

func TestExecute_NormalizeTrailingNewlines(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	withExcessNewlines := projectYAML + "\n\n\n"
	require.NoError(t, os.WriteFile("test-project.yaml", []byte(withExcessNewlines), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := ExecuteDevelopmentIteration(ctx, nil)
	require.NoError(t, err)

	content, err := os.ReadFile("test-project.yaml")
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(string(content), "\n"), "file should end with a newline")
	assert.False(t, strings.HasSuffix(string(content), "\n\n"), "file should have exactly one trailing newline")
}

func TestExecute_StagesFileWithChanges(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	// Commit a project file with excess trailing newlines so normalization produces a tracked change.
	withExcessNewlines := projectYAML + "\n\n\n"
	require.NoError(t, os.WriteFile("test-project.yaml", []byte(withExcessNewlines), 0644))
	for _, args := range [][]string{
		{"add", "test-project.yaml"},
		{"commit", "-m", "add project file"},
	} {
		c := exec.Command("git", args...)
		c.Dir = workDir
		require.NoError(t, c.Run())
	}

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := ExecuteDevelopmentIteration(ctx, nil)
	require.NoError(t, err)

	// Normalization should have changed and staged the file.
	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	cmd.Dir = workDir
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "test-project.yaml")
}

func TestExecute_DoesNotStageFileWithoutChanges(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	// Write the project file but do not commit it. Untracked files return no
	// diff against the index, so HasFileChanges returns false and StageFile is
	// never called regardless of mock-AI modifications.
	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := ExecuteDevelopmentIteration(ctx, nil)
	require.NoError(t, err)

	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	cmd.Dir = workDir
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.NotContains(t, string(out), "test-project.yaml")
}

func TestExecute_StartsServices(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	// Append a service to the existing .ralph/config.yaml.
	existing, err := os.ReadFile(filepath.Join(workDir, ".ralph", "config.yaml"))
	require.NoError(t, err)
	servicesCfg := string(existing) + "services:\n  - name: test-service\n    command: echo\n    args:\n      - test\n"
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ".ralph", "config.yaml"), []byte(servicesCfg), 0644))

	ctx := testutil.NewContext(
		testutil.WithProjectFile("test-project.yaml"),
		testutil.WithNoServices(false),
	)
	err = ExecuteDevelopmentIteration(ctx, nil)

	require.NoError(t, err)
}

func TestExecute_WritesBlockedOnAgentFailure(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	t.Setenv("RALPH_MOCK_AI_FAIL", "true")
	logger.SetVerbose(true)

	workDir := setupIterationTestRepo(t, "")
	t.Chdir(workDir)

	require.NoError(t, os.WriteFile("test-project.yaml", []byte(projectYAML), 0644))

	ctx := testutil.NewContext(testutil.WithProjectFile("test-project.yaml"))
	err := ExecuteDevelopmentIteration(ctx, nil)

	require.Error(t, err)

	blockedContent, err := os.ReadFile(filepath.Join(workDir, "blocked.md"))
	require.NoError(t, err)
	assert.Contains(t, string(blockedContent), "Blocked")
	assert.Contains(t, string(blockedContent), "opencode execution failed")
}
