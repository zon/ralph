package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zon/ralph/internal/context"
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

	ctx := context.NewContext(false, false, false, false)

	prompt, err := BuildDevelopPrompt(ctx, projectFile)
	if err != nil {
		t.Fatalf("BuildDevelopPrompt failed: %v", err)
	}

	// Verify prompt contains expected sections
	expectedSections := []string{
		"# Development Agent Context",
		"## Project Information",
		"## Recent Git History",
		"## Project Requirements",
		"## Instructions",
	}

	for _, section := range expectedSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("Prompt missing expected section: %s", section)
		}
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

	ctx := context.NewContext(false, false, false, false)

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
	ctx := context.NewContext(true, false, false, false)

	prompt, err := BuildDevelopPrompt(ctx, "dummy-file.yaml")
	if err != nil {
		t.Fatalf("BuildDevelopPrompt in dry-run failed: %v", err)
	}

	if prompt != "dry-run-prompt" {
		t.Errorf("Expected 'dry-run-prompt', got: %s", prompt)
	}
}

func TestBuildDevelopPrompt_MissingProjectFile(t *testing.T) {
	ctx := context.NewContext(false, false, false, false)

	_, err := BuildDevelopPrompt(ctx, "/nonexistent/project.yaml")
	if err == nil {
		t.Error("Expected error for missing project file, got nil")
	}
}

func TestGetDefaultInstructions(t *testing.T) {
	instructions := getDefaultInstructions()

	// Verify key points are in default instructions
	expectedPoints := []string{
		"ONLY WORK ON ONE REQUIREMENT",
		"passing: false",
		"Write tests",
		"report.md",
	}

	for _, point := range expectedPoints {
		if !strings.Contains(instructions, point) {
			t.Errorf("Default instructions missing expected content: %s", point)
		}
	}
}
