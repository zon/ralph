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
			name: "valid project with items",
			project: &Project{
				Name: "test-project",
				Requirements: []Requirement{
					{
						Category:    "backend",
						Description: "Test requirement",
						Items:       []string{"Item 1", "Item 2"},
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

func TestLoadProjectWithItems(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Project file matching ../slow-choice/projects/*.yaml format
	projectContent := `name: test-with-items
description: Test project with items

requirements:
  - category: backend
    description: User Authentication
    items:
      - User model with credentials
      - Password hashing capability
      - Login endpoint
    passing: false
  - category: testing
    description: Write tests
    items:
      - Authentication unit tests
      - Integration tests
    passing: false
`
	projectPath := filepath.Join(tmpDir, "test-items.yaml")
	if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to write project file: %v", err)
	}

	project, err := LoadProject(projectPath)
	if err != nil {
		t.Errorf("LoadProject() unexpected error: %v", err)
	}

	if project.Name != "test-with-items" {
		t.Errorf("LoadProject() Name = %s, want test-with-items", project.Name)
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
	if len(req1.Items) != 3 {
		t.Errorf("Requirement[0] Items length = %d, want 3", len(req1.Items))
	}
	if len(req1.Items) > 0 && req1.Items[0] != "User model with credentials" {
		t.Errorf("Requirement[0] Items[0] = %s, want 'User model with credentials'", req1.Items[0])
	}

	// Check second requirement
	req2 := project.Requirements[1]
	if req2.Category != "testing" {
		t.Errorf("Requirement[1] Category = %s, want testing", req2.Category)
	}
	if len(req2.Items) != 2 {
		t.Errorf("Requirement[1] Items length = %d, want 2", len(req2.Items))
	}
}

func TestLoadConfig_WithWorkDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.Mkdir(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph dir: %v", err)
	}

	configContent := `builds:
  - name: compile
    command: make
    args: [build]
    workDir: /tmp/myapp
services:
  - name: api
    command: ./api
    workDir: /tmp/myapp/bin
    port: 8080
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if len(cfg.Builds) != 1 {
		t.Fatalf("Builds length = %d, want 1", len(cfg.Builds))
	}
	if cfg.Builds[0].WorkDir != "/tmp/myapp" {
		t.Errorf("Build[0].WorkDir = %q, want /tmp/myapp", cfg.Builds[0].WorkDir)
	}

	if len(cfg.Services) != 1 {
		t.Fatalf("Services length = %d, want 1", len(cfg.Services))
	}
	if cfg.Services[0].WorkDir != "/tmp/myapp/bin" {
		t.Errorf("Service[0].WorkDir = %q, want /tmp/myapp/bin", cfg.Services[0].WorkDir)
	}
}

func TestLoadConfig_WorkDirOmitted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.Mkdir(ralphDir, 0755); err != nil {
		t.Fatalf("Failed to create .ralph dir: %v", err)
	}

	configContent := `builds:
  - name: compile
    command: make
services:
  - name: api
    command: ./api
    port: 8080
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if len(cfg.Builds) != 1 {
		t.Fatalf("Builds length = %d, want 1", len(cfg.Builds))
	}
	if cfg.Builds[0].WorkDir != "" {
		t.Errorf("Build[0].WorkDir = %q, want empty string", cfg.Builds[0].WorkDir)
	}

	if len(cfg.Services) != 1 {
		t.Fatalf("Services length = %d, want 1", len(cfg.Services))
	}
	if cfg.Services[0].WorkDir != "" {
		t.Errorf("Service[0].WorkDir = %q, want empty string", cfg.Services[0].WorkDir)
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

func TestLoadConfig_WithWorkflowConfig(t *testing.T) {
	// Create a temporary directory with workflow config
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

	// Write config file with workflow settings
	configContent := `maxIterations: 5
baseBranch: main
workflow:
  image:
    repository: ghcr.io/example/ralph-runner
    tag: v1.0.0
  configMaps:
    - name: app-config
    - name: shared-data
  secrets:
    - name: api-keys
    - name: database-creds
  env:
    LOG_LEVEL: debug
    APP_ENV: production
  context: my-k8s-cluster
  namespace: ralph-workflows
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	config, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() unexpected error: %v", err)
	}

	// Verify workflow image config
	if config.Workflow.Image.Repository != "ghcr.io/example/ralph-runner" {
		t.Errorf("Workflow.Image.Repository = %s, want ghcr.io/example/ralph-runner", config.Workflow.Image.Repository)
	}
	if config.Workflow.Image.Tag != "v1.0.0" {
		t.Errorf("Workflow.Image.Tag = %s, want v1.0.0", config.Workflow.Image.Tag)
	}

	// Verify configMaps
	if len(config.Workflow.ConfigMaps) != 2 {
		t.Errorf("Workflow.ConfigMaps length = %d, want 2", len(config.Workflow.ConfigMaps))
	}
	if len(config.Workflow.ConfigMaps) > 0 && config.Workflow.ConfigMaps[0].Name != "app-config" {
		t.Errorf("Workflow.ConfigMaps[0].Name = %s, want app-config", config.Workflow.ConfigMaps[0].Name)
	}
	if len(config.Workflow.ConfigMaps) > 1 && config.Workflow.ConfigMaps[1].Name != "shared-data" {
		t.Errorf("Workflow.ConfigMaps[1].Name = %s, want shared-data", config.Workflow.ConfigMaps[1].Name)
	}

	// Verify secrets
	if len(config.Workflow.Secrets) != 2 {
		t.Errorf("Workflow.Secrets length = %d, want 2", len(config.Workflow.Secrets))
	}
	if len(config.Workflow.Secrets) > 0 && config.Workflow.Secrets[0].Name != "api-keys" {
		t.Errorf("Workflow.Secrets[0].Name = %s, want api-keys", config.Workflow.Secrets[0].Name)
	}
	if len(config.Workflow.Secrets) > 1 && config.Workflow.Secrets[1].Name != "database-creds" {
		t.Errorf("Workflow.Secrets[1].Name = %s, want database-creds", config.Workflow.Secrets[1].Name)
	}

	// Verify environment variables
	if len(config.Workflow.Env) != 2 {
		t.Errorf("Workflow.Env length = %d, want 2", len(config.Workflow.Env))
	}
	if config.Workflow.Env["LOG_LEVEL"] != "debug" {
		t.Errorf("Workflow.Env[LOG_LEVEL] = %s, want debug", config.Workflow.Env["LOG_LEVEL"])
	}
	if config.Workflow.Env["APP_ENV"] != "production" {
		t.Errorf("Workflow.Env[APP_ENV] = %s, want production", config.Workflow.Env["APP_ENV"])
	}

	// Verify Kubernetes context and namespace
	if config.Workflow.Context != "my-k8s-cluster" {
		t.Errorf("Workflow.Context = %s, want my-k8s-cluster", config.Workflow.Context)
	}
	if config.Workflow.Namespace != "ralph-workflows" {
		t.Errorf("Workflow.Namespace = %s, want ralph-workflows", config.Workflow.Namespace)
	}
}

func TestLoadConfig_WithPartialWorkflowConfig(t *testing.T) {
	// Create a temporary directory with minimal workflow config
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

	// Write config file with only image repository
	configContent := `workflow:
  image:
    repository: my-registry/ralph
  context: dev-cluster
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	config, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() unexpected error: %v", err)
	}

	// Verify partial config loads correctly
	if config.Workflow.Image.Repository != "my-registry/ralph" {
		t.Errorf("Workflow.Image.Repository = %s, want my-registry/ralph", config.Workflow.Image.Repository)
	}
	if config.Workflow.Image.Tag != "" {
		t.Errorf("Workflow.Image.Tag = %s, want empty string", config.Workflow.Image.Tag)
	}
	if config.Workflow.Context != "dev-cluster" {
		t.Errorf("Workflow.Context = %s, want dev-cluster", config.Workflow.Context)
	}

	// Verify optional fields are empty/nil
	if len(config.Workflow.ConfigMaps) != 0 {
		t.Errorf("Workflow.ConfigMaps length = %d, want 0", len(config.Workflow.ConfigMaps))
	}
	if len(config.Workflow.Secrets) != 0 {
		t.Errorf("Workflow.Secrets length = %d, want 0", len(config.Workflow.Secrets))
	}
	if len(config.Workflow.Env) != 0 {
		t.Errorf("Workflow.Env length = %d, want 0", len(config.Workflow.Env))
	}
}

func TestLoadConfig_WithoutWorkflowConfig(t *testing.T) {
	// Create a temporary directory with config but no workflow section
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

	// Write config file without workflow section
	configContent := `maxIterations: 3
baseBranch: main
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	config, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() unexpected error: %v", err)
	}

	// Verify workflow config exists but is empty
	if config.Workflow.Image.Repository != "" {
		t.Errorf("Workflow.Image.Repository = %s, want empty string", config.Workflow.Image.Repository)
	}
	if config.Workflow.Image.Tag != "" {
		t.Errorf("Workflow.Image.Tag = %s, want empty string", config.Workflow.Image.Tag)
	}
	if config.Workflow.Context != "" {
		t.Errorf("Workflow.Context = %s, want empty string", config.Workflow.Context)
	}
	if config.Workflow.Namespace != "" {
		t.Errorf("Workflow.Namespace = %s, want empty string", config.Workflow.Namespace)
	}
}
