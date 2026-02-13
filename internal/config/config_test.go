package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateProject(t *testing.T) {
	tests := []struct {
		name    string
		project *Project
		wantErr bool
	}{
		{
			name: "valid project",
			project: &Project{
				Name: "test-project",
				Requirements: []Requirement{
					{ID: "req1", Passing: false},
				},
			},
			wantErr: false,
		},
		{
			name: "valid project with steps",
			project: &Project{
				Name: "test-project",
				Requirements: []Requirement{
					{
						Category:    "backend",
						Description: "Test requirement",
						Steps:       []string{"Step 1", "Step 2"},
						Passing:     false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			project: &Project{
				Name: "",
				Requirements: []Requirement{
					{ID: "req1", Passing: false},
				},
			},
			wantErr: true,
		},
		{
			name: "no requirements",
			project: &Project{
				Name:         "test-project",
				Requirements: []Requirement{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProject(tt.project)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckCompletion(t *testing.T) {
	tests := []struct {
		name         string
		project      *Project
		wantComplete bool
		wantPassing  int
		wantFailing  int
	}{
		{
			name: "all passing",
			project: &Project{
				Name: "test",
				Requirements: []Requirement{
					{Passing: true},
					{Passing: true},
				},
			},
			wantComplete: true,
			wantPassing:  2,
			wantFailing:  0,
		},
		{
			name: "mixed status",
			project: &Project{
				Name: "test",
				Requirements: []Requirement{
					{Passing: true},
					{Passing: false},
					{Passing: true},
				},
			},
			wantComplete: false,
			wantPassing:  2,
			wantFailing:  1,
		},
		{
			name: "all failing",
			project: &Project{
				Name: "test",
				Requirements: []Requirement{
					{Passing: false},
					{Passing: false},
				},
			},
			wantComplete: false,
			wantPassing:  0,
			wantFailing:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complete, passing, failing := CheckCompletion(tt.project)
			if complete != tt.wantComplete {
				t.Errorf("CheckCompletion() complete = %v, want %v", complete, tt.wantComplete)
			}
			if passing != tt.wantPassing {
				t.Errorf("CheckCompletion() passing = %v, want %v", passing, tt.wantPassing)
			}
			if failing != tt.wantFailing {
				t.Errorf("CheckCompletion() failing = %v, want %v", failing, tt.wantFailing)
			}
		})
	}
}

func TestUpdateRequirementStatus(t *testing.T) {
	project := &Project{
		Name: "test",
		Requirements: []Requirement{
			{ID: "req1", Passing: false},
			{ID: "req2", Passing: false},
		},
	}

	// Update existing requirement
	err := UpdateRequirementStatus(project, "req1", true)
	if err != nil {
		t.Errorf("UpdateRequirementStatus() unexpected error: %v", err)
	}
	if !project.Requirements[0].Passing {
		t.Error("UpdateRequirementStatus() did not update status")
	}

	// Try to update non-existent requirement
	err = UpdateRequirementStatus(project, "req999", true)
	if err == nil {
		t.Error("UpdateRequirementStatus() expected error for non-existent requirement")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Create a temporary directory without config
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	config, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() unexpected error: %v", err)
	}

	if config.MaxIterations != 10 {
		t.Errorf("LoadConfig() MaxIterations = %d, want 10", config.MaxIterations)
	}
	if config.BaseBranch != "main" {
		t.Errorf("LoadConfig() BaseBranch = %s, want main", config.BaseBranch)
	}
	if len(config.Services) != 0 {
		t.Errorf("LoadConfig() Services length = %d, want 0", len(config.Services))
	}
	if config.Instructions == "" {
		t.Error("LoadConfig() Instructions is empty, expected default instructions")
	}
	if !strings.Contains(config.Instructions, "## Instructions") {
		t.Error("LoadConfig() Instructions missing expected header")
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	// Create a temporary directory with config
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .ralph directory
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.Mkdir(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph dir: %v", err)
	}

	// Write config file
	configContent := `maxIterations: 5
baseBranch: develop
services:
  - name: test-service
    command: echo
    args: [hello]
    port: 8080
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Write custom instructions file
	instructionsContent := "Custom instructions for testing"
	instructionsPath := filepath.Join(ralphDir, "instructions.md")
	if err := os.WriteFile(instructionsPath, []byte(instructionsContent), 0644); err != nil {
		t.Fatalf("Failed to write instructions file: %v", err)
	}

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	config, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() unexpected error: %v", err)
	}

	if config.MaxIterations != 5 {
		t.Errorf("LoadConfig() MaxIterations = %d, want 5", config.MaxIterations)
	}
	if config.BaseBranch != "develop" {
		t.Errorf("LoadConfig() BaseBranch = %s, want develop", config.BaseBranch)
	}
	if len(config.Services) != 1 {
		t.Errorf("LoadConfig() Services length = %d, want 1", len(config.Services))
	}
	if len(config.Services) > 0 && config.Services[0].Name != "test-service" {
		t.Errorf("LoadConfig() Service name = %s, want test-service", config.Services[0].Name)
	}
	if config.Instructions != instructionsContent {
		t.Errorf("LoadConfig() Instructions = %s, want %s", config.Instructions, instructionsContent)
	}
}

func TestLoadProject(t *testing.T) {
	// Create a temporary project file
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectContent := `name: test-project
description: A test project
requirements:
  - id: req1
    name: First requirement
    passing: false
  - id: req2
    name: Second requirement
    passing: true
`
	projectPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to write project file: %v", err)
	}

	project, err := LoadProject(projectPath)
	if err != nil {
		t.Errorf("LoadProject() unexpected error: %v", err)
	}

	if project.Name != "test-project" {
		t.Errorf("LoadProject() Name = %s, want test-project", project.Name)
	}
	if project.Description != "A test project" {
		t.Errorf("LoadProject() Description = %s, want 'A test project'", project.Description)
	}
	if len(project.Requirements) != 2 {
		t.Errorf("LoadProject() Requirements length = %d, want 2", len(project.Requirements))
	}
}

func TestLoadProjectWithSteps(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Project file matching ../slow-choice/projects/*.yaml format
	projectContent := `name: test-with-steps
description: Test project with steps

requirements:
  - category: backend
    description: User Authentication
    steps:
      - Create User model
      - Add password hashing
      - Implement login endpoint
    passing: false
  - category: testing
    description: Write tests
    steps:
      - Unit tests for auth
      - Integration tests
    passing: false
`
	projectPath := filepath.Join(tmpDir, "test-steps.yaml")
	if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to write project file: %v", err)
	}

	project, err := LoadProject(projectPath)
	if err != nil {
		t.Errorf("LoadProject() unexpected error: %v", err)
	}

	if project.Name != "test-with-steps" {
		t.Errorf("LoadProject() Name = %s, want test-with-steps", project.Name)
	}
	if len(project.Requirements) != 2 {
		t.Fatalf("LoadProject() Requirements length = %d, want 2", len(project.Requirements))
	}

	// Check first requirement
	req1 := project.Requirements[0]
	if req1.Category != "backend" {
		t.Errorf("Requirement[0] Category = %s, want backend", req1.Category)
	}
	if req1.Description != "User Authentication" {
		t.Errorf("Requirement[0] Description = %s, want 'User Authentication'", req1.Description)
	}
	if len(req1.Steps) != 3 {
		t.Errorf("Requirement[0] Steps length = %d, want 3", len(req1.Steps))
	}
	if len(req1.Steps) > 0 && req1.Steps[0] != "Create User model" {
		t.Errorf("Requirement[0] Steps[0] = %s, want 'Create User model'", req1.Steps[0])
	}

	// Check second requirement
	req2 := project.Requirements[1]
	if req2.Category != "testing" {
		t.Errorf("Requirement[1] Category = %s, want testing", req2.Category)
	}
	if len(req2.Steps) != 2 {
		t.Errorf("Requirement[1] Steps length = %d, want 2", len(req2.Steps))
	}
}

func TestSaveProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	project := &Project{
		Name:        "test-project",
		Description: "Test description",
		Requirements: []Requirement{
			{ID: "req1", Name: "Requirement 1", Passing: true},
		},
	}

	projectPath := filepath.Join(tmpDir, "project.yaml")
	err = SaveProject(projectPath, project)
	if err != nil {
		t.Errorf("SaveProject() unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("SaveProject() did not create file")
	}

	// Load it back and verify
	loaded, err := LoadProject(projectPath)
	if err != nil {
		t.Errorf("LoadProject() after save unexpected error: %v", err)
	}
	if loaded.Name != project.Name {
		t.Errorf("Loaded project name = %s, want %s", loaded.Name, project.Name)
	}
}
