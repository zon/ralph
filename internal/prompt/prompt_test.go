package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/testutil"
)

func TestBuildDevelopPrompt(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
description: A test project
requirements:
  - category: Feature
    description: Implement feature X
    passing: false
  - category: Bug Fix
    description: Fix bug Y
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Change to temp directory to avoid needing actual git repo
	t.Chdir(tmpDir)

	ctx := testutil.NewContext()

	selectedReq := `- category: Feature
  description: Implement feature X
  passing: false`

	prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	// Verify prompt contains expected sections
	expectedSections := []string{
		"# Development Agent Context",
		"## Project Information",
		"## Selected Requirement",
		"## Instructions",
	}

	for _, section := range expectedSections {
		assert.True(t, strings.Contains(prompt, section), "Prompt missing expected section: %s", section)
	}

	// Without a git repo, GetCurrentBranch fails and no commit history is included
	assert.False(t, strings.Contains(prompt, "## Recent Git History"), "Prompt should NOT contain 'Recent Git History' section without a git repo")

	// Verify selected requirement is included
	assert.True(t, strings.Contains(prompt, "Implement feature X"), "Prompt does not contain selected requirement")

	// Verify default instructions are included (since no .ralph/instructions.md exists)
	assert.True(t, strings.Contains(prompt, "Implement the selected requirement"), "Prompt does not contain default instructions")
}

func TestBuildDevelopPrompt_WithCustomInstructions(t *testing.T) {
	// Create a temporary project structure with custom instructions
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
description: A test project
requirements:
  - category: Feature
    description: Implement feature X
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	customInstructions := "Custom instruction: Do this specific thing"
	instructionsFile := filepath.Join(ralphDir, "instructions.md")
	if err := os.WriteFile(instructionsFile, []byte(customInstructions), 0644); err != nil {
		t.Fatalf("Failed to create instructions file: %v", err)
	}

	t.Chdir(tmpDir)

	ctx := testutil.NewContext()

	selectedReq := "- category: Feature\n  description: Implement feature X\n  passing: false"

	prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	// Verify custom instructions are included
	assert.True(t, strings.Contains(prompt, customInstructions), "Prompt does not contain custom instructions")
}

func TestBuildDevelopPrompt_NoGitRepo(t *testing.T) {
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

	ctx := testutil.NewContext()

	selectedReq := "- description: Test requirement\n  passing: false"

	prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.True(t, strings.Contains(prompt, "Development Agent Context"), "Prompt should contain 'Development Agent Context' header")
	assert.True(t, strings.Contains(prompt, "Test requirement"), "Prompt should contain selected requirement")
}

func TestBuildDevelopPrompt_MissingProjectFile(t *testing.T) {
	ctx := testutil.NewContext()

	_, err := BuildDevelopPrompt(ctx, "/nonexistent/project.yaml", "some requirement")
	require.Error(t, err, "Expected error for missing project file")
}

func TestBuildServiceFixPrompt(t *testing.T) {
	ctx := testutil.NewContext()
	svcErr := fmt.Errorf("failed to start service test-service: connection refused")

	svc := config.Service{
		Name:    "test-service",
		Command: "myapp",
		Args:    []string{"--port", "8080"},
		Port:    8080,
	}
	prompt := BuildServiceFixPrompt(ctx, svc, svcErr)

	// Verify error message is present
	assert.True(t, strings.Contains(prompt, "Service Startup Failed"), "Prompt does not contain service failure header")
	assert.True(t, strings.Contains(prompt, "failed to start service test-service"), "Prompt does not contain service failure details")

	// Verify service details are present
	assert.True(t, strings.Contains(prompt, "test-service"), "Prompt does not contain service name")
	assert.True(t, strings.Contains(prompt, "myapp --port 8080"), "Prompt does not contain start command")
	assert.True(t, strings.Contains(prompt, "port 8080"), "Prompt does not contain health check port")

	// Verify full dev prompt sections are absent
	assert.False(t, strings.Contains(prompt, "## Project Requirements"), "Service fix prompt should not contain project requirements")
	assert.False(t, strings.Contains(prompt, "## Recent Git History"), "Service fix prompt should not contain git history")

	// Verify fix-service instructions are present
	assert.True(t, strings.Contains(prompt, "report.md"), "Service fix prompt should contain report.md instruction")
}

func TestBuildServiceFixPrompt_NoPort(t *testing.T) {
	ctx := testutil.NewContext()
	svcErr := fmt.Errorf("failed to start service worker: exit status 1")

	svc := config.Service{
		Name:    "worker",
		Command: "worker",
		Args:    []string{"--config", "worker.yaml"},
	}
	prompt := BuildServiceFixPrompt(ctx, svc, svcErr)

	assert.True(t, strings.Contains(prompt, "worker --config worker.yaml"), "Prompt does not contain start command")
	assert.False(t, strings.Contains(prompt, "Health check"), "Prompt should not contain health check when no port configured")
}

func TestBuildDevelopPrompt_WithInstructionsFlag(t *testing.T) {
	tests := []struct {
		name                string
		instructionsContent string
		wantContains        []string
		wantNotContains     []string
	}{
		{
			name:                "instructions file overrides default instructions",
			instructionsContent: "Custom webhook instructions: handle PR comments",
			wantContains:        []string{"Custom webhook instructions: handle PR comments"},
			wantNotContains:     []string{"ONLY WORK ON ONE REQUIREMENT"},
		},
		{
			name:                "instructions file overrides custom .ralph/instructions.md",
			instructionsContent: "Override from --instructions flag",
			wantContains:        []string{"Override from --instructions flag"},
			wantNotContains:     []string{"ONLY WORK ON ONE REQUIREMENT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			projectFile := filepath.Join(tmpDir, "test-project.yaml")
			projectContent := `name: Test Project
requirements:
  - description: Feature 1
    passing: false
`
			if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
				t.Fatalf("Failed to create test project file: %v", err)
			}

			// Write the instructions file
			instructionsFile := filepath.Join(tmpDir, "webhook-instructions.md")
			if err := os.WriteFile(instructionsFile, []byte(tt.instructionsContent), 0644); err != nil {
				t.Fatalf("Failed to create instructions file: %v", err)
			}

			t.Chdir(tmpDir)

			ctx := testutil.NewContext(testutil.WithInstructions(instructionsFile))

			selectedReq := "- description: Feature 1\n  passing: false"
			prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
			require.NoError(t, err, "BuildDevelopPrompt failed")

			for _, want := range tt.wantContains {
				assert.True(t, strings.Contains(prompt, want), "Prompt missing expected content %q", want)
			}
			for _, notWant := range tt.wantNotContains {
				assert.False(t, strings.Contains(prompt, notWant), "Prompt should not contain %q", notWant)
			}
		})
	}
}

func TestBuildDevelopPrompt_InstructionsFlagOverridesRalphInstructions(t *testing.T) {
	// Verify --instructions flag takes precedence over .ralph/instructions.md
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
requirements:
  - description: Feature 1
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Create .ralph/instructions.md with its own content
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	ralphInstructions := "Instructions from .ralph/instructions.md"
	if err := os.WriteFile(filepath.Join(ralphDir, "instructions.md"), []byte(ralphInstructions), 0644); err != nil {
		t.Fatalf("Failed to create .ralph/instructions.md: %v", err)
	}

	// Create a separate --instructions file
	overrideInstructions := "Instructions from --instructions flag (should win)"
	instructionsFile := filepath.Join(tmpDir, "override.md")
	if err := os.WriteFile(instructionsFile, []byte(overrideInstructions), 0644); err != nil {
		t.Fatalf("Failed to create override instructions file: %v", err)
	}

	t.Chdir(tmpDir)

	ctx := testutil.NewContext(testutil.WithInstructions(instructionsFile))

	selectedReq := "- description: Feature 1\n  passing: false"
	prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	assert.True(t, strings.Contains(prompt, overrideInstructions), "Prompt should contain --instructions file content, got:\n%s", prompt)
	assert.False(t, strings.Contains(prompt, ralphInstructions), "Prompt should NOT contain .ralph/instructions.md content when --instructions flag is set")
}

func TestBuildDevelopPrompt_InstructionsFlagMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
requirements:
  - description: Feature 1
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	t.Chdir(tmpDir)

	// Point to a non-existent file
	ctx := testutil.NewContext(testutil.WithInstructions("/nonexistent/instructions.md"))

	_, err := BuildDevelopPrompt(ctx, projectFile, "- description: Feature 1\n  passing: false")
	require.Error(t, err, "Expected error when instructions file does not exist")
	assert.True(t, strings.Contains(err.Error(), "failed to read instructions file"), "Expected 'failed to read instructions file' error, got: %v", err)
}

func TestBuildDevelopPrompt_WithoutNote(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
requirements:
  - description: Feature 1
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	t.Chdir(tmpDir)

	// Create context without notes
	ctx := testutil.NewContext()

	selectedReq := "- description: Feature 1\n  passing: false"
	prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	// Verify prompt does NOT contain system notes section when context has no notes
	assert.False(t, strings.Contains(prompt, "## System Notes"), "Prompt should not contain 'System Notes' section when context has no notes")
}

func TestBuildDevelopPrompt_WithNote(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
requirements:
  - description: Feature 1
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	t.Chdir(tmpDir)

	// Create context with notes added via AddNote
	ctx := testutil.NewContext()
	ctx.AddNote("This is a test note for the agent")
	ctx.AddNote("Another note with important information")

	selectedReq := "- description: Feature 1\n  passing: false"
	prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	// Verify prompt contains System Notes section when context has notes
	assert.True(t, strings.Contains(prompt, "## System Notes"), "Prompt should contain 'System Notes' section when context has notes")

	// Verify the notes are included in the output
	assert.True(t, strings.Contains(prompt, "This is a test note for the agent"), "Prompt should contain the first note")
	assert.True(t, strings.Contains(prompt, "Another note with important information"), "Prompt should contain the second note")
}

func TestBuildDevelopPrompt_CommitLogWhenBranchDiffers(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
requirements:
  - description: Feature 1
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Create .ralph/config.yaml with a defaultBranch
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	configContent := "defaultBranch: main\n"
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Chdir(tmpDir)

	// Without a git repo, GetCurrentBranch fails and no commit history is included
	// regardless of defaultBranch setting
	ctx := testutil.NewContext()

	selectedReq := "- description: Feature 1\n  passing: false"
	prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	// Verify prompt does NOT contain Recent Git History section when there's no git repo
	assert.False(t, strings.Contains(prompt, "## Recent Git History"), "Prompt should NOT contain 'Recent Git History' section without a git repo")
}

func TestBuildDevelopPrompt_NoCommitLogWhenBranchMatches(t *testing.T) {
	// Create a temporary project file
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
requirements:
  - description: Feature 1
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	// Create .ralph/config.yaml with a defaultBranch
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}
	configContent := "defaultBranch: main\n"
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	t.Chdir(tmpDir)

	// Without a git repo, GetCurrentBranch fails and no commit history is included
	ctx := testutil.NewContext()

	selectedReq := "- description: Feature 1\n  passing: false"
	prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
	require.NoError(t, err, "BuildDevelopPrompt failed")

	// Verify prompt does NOT contain Recent Git History section
	assert.False(t, strings.Contains(prompt, "## Recent Git History"), "Prompt should NOT contain 'Recent Git History' section when current branch equals base branch")
}

func TestBuildPickPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
description: A test project
requirements:
  - category: Feature
    description: Implement feature X
    passing: false
  - category: Bug Fix
    description: Fix bug Y
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	t.Chdir(tmpDir)

	ctx := testutil.NewContext()

	pickedReqPath := filepath.Join(tmpDir, "picked-requirement.yaml")
	prompt, err := BuildPickPrompt(ctx, projectFile, pickedReqPath)
	require.NoError(t, err, "BuildPickPrompt failed")

	// Verify prompt contains expected sections
	expectedSections := []string{
		"# Requirement Picker Agent Context",
		"## Project Information",
		"## Project Requirements",
		"## Instructions",
	}

	for _, section := range expectedSections {
		assert.True(t, strings.Contains(prompt, section), "Prompt missing expected section: %s", section)
	}

	// Without a git repo, GetCurrentBranch fails and no commit history is included
	assert.False(t, strings.Contains(prompt, "## Recent Git History"), "Prompt should NOT contain 'Recent Git History' section without a git repo")

	// Verify project content is included
	assert.True(t, strings.Contains(prompt, "Test Project"), "Prompt does not contain project name")
	assert.True(t, strings.Contains(prompt, "Implement feature X"), "Prompt does not contain project requirements")

	// Verify the prompt contains the exact absolute path — this is the critical regression test:
	// the AI must be told exactly where to write the file so it doesn't default to CWD when
	// the project file lives in a subdirectory (e.g. projects/foo.yaml vs repo root).
	assert.True(t, strings.Contains(prompt, pickedReqPath), "Prompt must contain the absolute path of picked-requirement.yaml so the AI writes it to the correct location, got:\n%s", prompt)
}

func TestBuildPickPrompt_MissingProjectFile(t *testing.T) {
	ctx := testutil.NewContext()

	_, err := BuildPickPrompt(ctx, "/nonexistent/project.yaml", "/tmp/picked-requirement.yaml")
	require.Error(t, err, "Expected error for missing project file")
	assert.Contains(t, err.Error(), "failed to read project file")
}

func TestBuildPickPrompt_WithNotes(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
requirements:
  - description: Feature 1
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	t.Chdir(tmpDir)

	ctx := testutil.NewContext()
	ctx.AddNote("This is a test note for the picker agent")

	pickedReqPath := filepath.Join(tmpDir, "picked-requirement.yaml")
	prompt, err := BuildPickPrompt(ctx, projectFile, pickedReqPath)
	require.NoError(t, err, "BuildPickPrompt failed")

	// Verify prompt contains System Notes section when context has notes
	assert.True(t, strings.Contains(prompt, "## System Notes"), "Prompt should contain 'System Notes' section when context has notes")
	assert.True(t, strings.Contains(prompt, "This is a test note for the picker agent"), "Prompt should contain the note")
}

func TestBuildPickPrompt_WithoutNotes(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
requirements:
  - description: Feature 1
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	t.Chdir(tmpDir)

	ctx := testutil.NewContext()

	pickedReqPath := filepath.Join(tmpDir, "picked-requirement.yaml")
	prompt, err := BuildPickPrompt(ctx, projectFile, pickedReqPath)
	require.NoError(t, err, "BuildPickPrompt failed")

	// Verify prompt does NOT contain system notes section when context has no notes
	assert.False(t, strings.Contains(prompt, "## System Notes"), "Prompt should not contain 'System Notes' section when context has no notes")
}

func TestBuildDevelopPrompt_WithSelectedRequirement(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")
	projectContent := `name: Test Project
description: A test project
requirements:
  - category: Feature
    description: Implement feature X
    passing: false
  - category: Bug Fix
    description: Fix bug Y
    passing: false
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	t.Chdir(tmpDir)

	ctx := testutil.NewContext()

	selectedReq := `  - category: Feature
    description: Implement feature X
    passing: false`

	prompt, err := BuildDevelopPrompt(ctx, projectFile, selectedReq)
	require.NoError(t, err, "BuildDevelopPrompt with selectedRequirement failed")

	// Verify selected requirement is included inline
	assert.True(t, strings.Contains(prompt, "Implement feature X"), "Prompt does not contain selected requirement")

	// Verify project file path is included
	assert.True(t, strings.Contains(prompt, projectFile), "Prompt does not contain project file path")

	// Verify instructions are still included
	assert.True(t, strings.Contains(prompt, "Implement the selected requirement"), "Prompt does not contain instructions")
}
