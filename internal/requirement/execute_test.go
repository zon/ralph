package requirement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	require.NoError(t, err, "Execute failed in dry-run mode")
}

func TestExecute_InvalidProjectFile(t *testing.T) {
	ctx := testutil.NewContext(testutil.WithProjectFile("/nonexistent/project.yaml"))

	// Test with non-existent file
	err := Execute(ctx, nil)
	require.Error(t, err, "Expected error for non-existent project file")
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

	require.Error(t, err, "Expected error for invalid YAML")
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

	require.Error(t, err, "Expected error for project with no requirements")
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

	if err := os.WriteFile(projectFile, []byte(projectYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	blockedPath := filepath.Join(tmpDir, "blocked.md")
	blockedContent := "Agent is blocked due to previous error"
	if err := os.WriteFile(blockedPath, []byte(blockedContent), 0644); err != nil {
		t.Fatalf("Failed to create blocked.md file: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.Error(t, err, "Expected error when blocked.md exists")

	assert.True(t, strings.Contains(err.Error(), "blocked"), "Expected error message to mention 'blocked', got: %v", err)
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

	if err := os.WriteFile(projectFile, []byte(projectYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	blockedPath := filepath.Join(tmpDir, "blocked.md")
	blockedContent := "This is the blocked reason from blocked.md"
	if err := os.WriteFile(blockedPath, []byte(blockedContent), 0644); err != nil {
		t.Fatalf("Failed to create blocked.md file: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.Error(t, err, "Expected error when blocked.md exists")

	assert.True(t, strings.Contains(err.Error(), blockedContent), "Expected error message to contain blocked.md contents, got: %v", err)
}

func TestExecute_NoBlockedMD(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.NoError(t, err, "Execute failed without blocked.md")
}

func TestExecute_NormalizeTrailingNewlines(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAMLWithExcessNewlines := "name: Test Project\ndescription: Test project\nrequirements:\n  - id: req1\n    description: Test requirement\n    passing: false\n\n\n\n"

	if err := os.WriteFile(projectFile, []byte(projectYAMLWithExcessNewlines), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	err := Execute(ctx, nil)

	require.NoError(t, err, "Execute failed")

	content, err := os.ReadFile(projectFile)
	require.NoError(t, err, "Failed to read project file")

	assert.True(t, strings.HasSuffix(string(content), "\n"), "Expected file to end with a newline")
	assert.False(t, strings.HasSuffix(string(content), "\n\n"), "Expected file to have exactly one trailing newline, found excess")
}

func TestExecute_StagesFileWithChanges(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithDryRun(true),
	)

	err := Execute(ctx, nil)

	require.NoError(t, err, "Execute failed")
}

func TestExecute_DoesNotStageFileWithoutChanges(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithDryRun(true),
	)

	err := Execute(ctx, nil)

	require.NoError(t, err, "Execute failed")
}

func TestExecute_StartsServices(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	configYAML := `services:
  - name: test-service
    command: echo
    args:
      - "test"
`
	configFile := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Chdir(tmpDir)

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithNoServices(false),
		testutil.WithDryRun(true),
	)

	err := Execute(ctx, nil)

	require.NoError(t, err, "Execute failed")
}

func TestExecute_StartsServicesDryRunMode(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectYAML := `name: Test Project
description: Test project
requirements:
  - id: req1
    description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectYAML), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	configYAML := `services:
  - name: test-service
    command: echo
    args:
      - "test"
`
	configFile := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Chdir(tmpDir)

	ctx := testutil.NewContext(
		testutil.WithProjectFile(projectFile),
		testutil.WithNoServices(false),
		testutil.WithDryRun(true),
	)

	err := Execute(ctx, nil)

	require.NoError(t, err, "Execute failed in dry-run mode")
}
