package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	ctx := testutil.NewContext()

	prompt, err := BuildDevelopPrompt(ctx, projectFile)
	if err != nil {
		t.Fatalf("BuildDevelopPrompt failed: %v", err)
	}

	// Verify prompt contains expected sections
	expectedSections := []string{
		"# Development Agent Context",
		"## Project Information",
		"## Project Requirements",
		"## Instructions",
	}

	for _, section := range expectedSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("Prompt missing expected section: %s", section)
		}
	}

	// In dry-run mode, GetCurrentBranch returns "dry-run-branch" (not "main"),
	// so Recent Git History section will be included with dummy commits.
	// This is expected behavior for dry-run mode.
	if !strings.Contains(prompt, "## Recent Git History") {
		t.Error("Prompt should contain 'Recent Git History' section in dry-run mode")
	}

	// Verify project content is included
	if !strings.Contains(prompt, "Test Project") {
		t.Error("Prompt does not contain project name")
	}

	if !strings.Contains(prompt, "Implement feature X") {
		t.Error("Prompt does not contain project requirements")
	}

	// Verify default instructions are included (since no .ralph/instructions.md exists)
	if !strings.Contains(prompt, "ONLY WORK ON ONE REQUIREMENT") {
		t.Error("Prompt does not contain default instructions")
	}
}

func TestBuildDevelopPrompt_WithCustomInstructions(t *testing.T) {
	// Create a temporary project structure with custom instructions
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

	// Create .ralph directory and custom instructions
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph directory: %v", err)
	}

	customInstructions := "Custom instruction: Do this specific thing"
	instructionsFile := filepath.Join(ralphDir, "instructions.md")
	if err := os.WriteFile(instructionsFile, []byte(customInstructions), 0644); err != nil {
		t.Fatalf("Failed to create instructions file: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	ctx := testutil.NewContext()

	prompt, err := BuildDevelopPrompt(ctx, projectFile)
	if err != nil {
		t.Fatalf("BuildDevelopPrompt failed: %v", err)
	}

	// Verify custom instructions are included
	if !strings.Contains(prompt, customInstructions) {
		t.Error("Prompt does not contain custom instructions")
	}

	// Verify default instructions are NOT included
	if strings.Contains(prompt, "ONLY WORK ON ONE REQUIREMENT") {
		t.Error("Prompt should not contain default instructions when custom file exists")
	}
}

func TestBuildDevelopPrompt_DryRun(t *testing.T) {
	// Create a temporary project file for dry-run testing
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "test-project.yaml")

	projectContent := `name: Test Project
description: A test project in dry-run mode
requirements:
  - description: Test requirement
    passing: false
`

	if err := os.WriteFile(projectFile, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create test project file: %v", err)
	}

	ctx := testutil.NewContext()

	prompt, err := BuildDevelopPrompt(ctx, projectFile)
	if err != nil {
		t.Fatalf("BuildDevelopPrompt in dry-run failed: %v", err)
	}

	// In dry-run mode, the prompt should still be built (not a dummy value)
	// Verify it contains expected sections
	if !strings.Contains(prompt, "Development Agent Context") {
		t.Error("Prompt should contain 'Development Agent Context' header even in dry-run")
	}

	if !strings.Contains(prompt, "Test Project") {
		t.Error("Prompt should contain project name even in dry-run")
	}
}

func TestBuildDevelopPrompt_MissingProjectFile(t *testing.T) {
	ctx := testutil.NewContext()

	_, err := BuildDevelopPrompt(ctx, "/nonexistent/project.yaml")
	if err == nil {
		t.Error("Expected error for missing project file, got nil")
	}
}

func TestBuildDevelopPrompt_WithNote(t *testing.T) {
	// Create a temporary project file
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

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create context with a note
	ctx := testutil.NewContext()
	ctx.AddNote("# Service Startup Failed\n\nfailed to start service test-service: connection refused\n\nServices are required. Fix this before proceeding.")

	prompt, err := BuildDevelopPrompt(ctx, projectFile)
	if err != nil {
		t.Fatalf("BuildDevelopPrompt failed: %v", err)
	}

	// Verify prompt contains the system notes section
	if !strings.Contains(prompt, "## System Notes") {
		t.Error("Prompt missing 'System Notes' section when context has notes")
	}

	// Verify note content is included
	if !strings.Contains(prompt, "Service Startup Failed") {
		t.Error("Prompt does not contain note content")
	}

	if !strings.Contains(prompt, "failed to start service test-service") {
		t.Error("Prompt does not contain service failure details from note")
	}

	if !strings.Contains(prompt, "Services are required. Fix this before proceeding.") {
		t.Error("Prompt does not contain service requirement message from note")
	}
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

			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer os.Chdir(oldWd)
			os.Chdir(tmpDir)

			ctx := testutil.NewContext(testutil.WithInstructions(instructionsFile))

			prompt, err := BuildDevelopPrompt(ctx, projectFile)
			if err != nil {
				t.Fatalf("BuildDevelopPrompt failed: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(prompt, want) {
					t.Errorf("Prompt missing expected content %q", want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(prompt, notWant) {
					t.Errorf("Prompt should not contain %q", notWant)
				}
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

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	ctx := testutil.NewContext(testutil.WithInstructions(instructionsFile))

	prompt, err := BuildDevelopPrompt(ctx, projectFile)
	if err != nil {
		t.Fatalf("BuildDevelopPrompt failed: %v", err)
	}

	if !strings.Contains(prompt, overrideInstructions) {
		t.Errorf("Prompt should contain --instructions file content, got:\n%s", prompt)
	}
	if strings.Contains(prompt, ralphInstructions) {
		t.Errorf("Prompt should NOT contain .ralph/instructions.md content when --instructions flag is set")
	}
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

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Point to a non-existent file
	ctx := testutil.NewContext(testutil.WithInstructions("/nonexistent/instructions.md"))

	_, err = BuildDevelopPrompt(ctx, projectFile)
	if err == nil {
		t.Error("Expected error when instructions file does not exist, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read instructions file") {
		t.Errorf("Expected 'failed to read instructions file' error, got: %v", err)
	}
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

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create context without notes
	ctx := testutil.NewContext()

	prompt, err := BuildDevelopPrompt(ctx, projectFile)
	if err != nil {
		t.Fatalf("BuildDevelopPrompt failed: %v", err)
	}

	// Verify prompt does NOT contain system notes section when context has no notes
	if strings.Contains(prompt, "## System Notes") {
		t.Error("Prompt should not contain 'System Notes' section when context has no notes")
	}
}
