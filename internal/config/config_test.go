package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
			assert.Equal(t, tt.wantComplete, complete)
			assert.Equal(t, tt.wantPassing, passing)
			assert.Equal(t, tt.wantFailing, failing)
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
	require.NoError(t, err, "UpdateRequirementStatus() unexpected error")
	assert.True(t, project.Requirements[0].Passing, "UpdateRequirementStatus() did not update status")

	// Try to update non-existent requirement
	err = UpdateRequirementStatus(project, "req999", true)
	require.Error(t, err, "UpdateRequirementStatus() expected error for non-existent requirement")
}

func TestLoadConfig_Defaults(t *testing.T) {
	tmpDir := t.TempDir()

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, 10, config.MaxIterations)
	assert.Equal(t, "main", config.BaseBranch)
	assert.Empty(t, config.Services)
	assert.NotEmpty(t, config.Instructions, "LoadConfig() Instructions is empty, expected default instructions")
	assert.True(t, strings.Contains(config.Instructions, "## Instructions"), "LoadConfig() Instructions missing expected header")
}

func TestLoadConfig_FromFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Chdir(tmpDir)

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

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
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Write custom instructions file
	instructionsContent := "Custom instructions for testing"
	instructionsPath := filepath.Join(ralphDir, "instructions.md")
	require.NoError(t, os.WriteFile(instructionsPath, []byte(instructionsContent), 0644))

	// Change to temp directory
	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, 5, config.MaxIterations)
	assert.Equal(t, "develop", config.BaseBranch)
	assert.Len(t, config.Services, 1)
	assert.Equal(t, "test-service", config.Services[0].Name)
	assert.Equal(t, instructionsContent, config.Instructions)
}

func TestLoadProject(t *testing.T) {
	tmpDir := t.TempDir()

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
	require.NoError(t, os.WriteFile(projectPath, []byte(projectContent), 0644))

	project, err := LoadProject(projectPath)
	require.NoError(t, err, "LoadProject() unexpected error")

	assert.Equal(t, "test-project", project.Name)
	assert.Equal(t, "A test project", project.Description)
	assert.Len(t, project.Requirements, 2)
}

func TestLoadProjectWithItems(t *testing.T) {
	tmpDir := t.TempDir()

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
	require.NoError(t, os.WriteFile(projectPath, []byte(projectContent), 0644))

	project, err := LoadProject(projectPath)
	require.NoError(t, err, "LoadProject() unexpected error")

	assert.Equal(t, "test-with-items", project.Name)
	require.Len(t, project.Requirements, 2)

	// Check first requirement
	req1 := project.Requirements[0]
	assert.Equal(t, "backend", req1.Category)
	assert.Equal(t, "User Authentication", req1.Description)
	require.Len(t, req1.Items, 3)
	assert.Equal(t, "User model with credentials", req1.Items[0])

	// Check second requirement
	req2 := project.Requirements[1]
	assert.Equal(t, "testing", req2.Category)
	assert.Len(t, req2.Items, 2)
}

func TestLoadConfig_WithWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `before:
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
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	require.Len(t, cfg.Before, 1)
	assert.Equal(t, "/tmp/myapp", cfg.Before[0].WorkDir)

	require.Len(t, cfg.Services, 1)
	assert.Equal(t, "/tmp/myapp/bin", cfg.Services[0].WorkDir)
}

func TestLoadConfig_WorkDirOmitted(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `before:
  - name: compile
    command: make
services:
  - name: api
    command: ./api
    port: 8080
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	require.Len(t, cfg.Before, 1)
	assert.Equal(t, "", cfg.Before[0].WorkDir)

	require.Len(t, cfg.Services, 1)
	assert.Equal(t, "", cfg.Services[0].WorkDir)
}

func TestSaveProject(t *testing.T) {
	tmpDir := t.TempDir()

	project := &Project{
		Name:        "test-project",
		Description: "Test description",
		Requirements: []Requirement{
			{ID: "req1", Name: "Requirement 1", Passing: true},
		},
	}

	projectPath := filepath.Join(tmpDir, "project.yaml")
	require.NoError(t, SaveProject(projectPath, project), "SaveProject() unexpected error")

	// Verify file was created
	_, err := os.Stat(projectPath)
	require.NoError(t, err, "SaveProject() did not create file")

	// Load it back and verify
	loaded, err := LoadProject(projectPath)
	require.NoError(t, err, "LoadProject() after save unexpected error")
	assert.Equal(t, project.Name, loaded.Name)
}

func TestLoadConfig_WithWorkflowConfig(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

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
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	// Verify workflow image config
	assert.Equal(t, "ghcr.io/example/ralph-runner", config.Workflow.Image.Repository)
	assert.Equal(t, "v1.0.0", config.Workflow.Image.Tag)

	// Verify configMaps
	require.Len(t, config.Workflow.ConfigMaps, 2)
	assert.Equal(t, "app-config", config.Workflow.ConfigMaps[0].Name)
	assert.Equal(t, "shared-data", config.Workflow.ConfigMaps[1].Name)

	// Verify secrets
	require.Len(t, config.Workflow.Secrets, 2)
	assert.Equal(t, "api-keys", config.Workflow.Secrets[0].Name)
	assert.Equal(t, "database-creds", config.Workflow.Secrets[1].Name)

	// Verify environment variables
	require.Len(t, config.Workflow.Env, 2)
	assert.Equal(t, "debug", config.Workflow.Env["LOG_LEVEL"])
	assert.Equal(t, "production", config.Workflow.Env["APP_ENV"])

	// Verify Kubernetes context and namespace
	assert.Equal(t, "my-k8s-cluster", config.Workflow.Context)
	assert.Equal(t, "ralph-workflows", config.Workflow.Namespace)
}

func TestLoadConfig_WithPartialWorkflowConfig(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `workflow:
  image:
    repository: my-registry/ralph
  context: dev-cluster
  `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	// Verify partial config loads correctly
	assert.Equal(t, "my-registry/ralph", config.Workflow.Image.Repository)
	assert.Equal(t, "", config.Workflow.Image.Tag)
	assert.Equal(t, "dev-cluster", config.Workflow.Context)

	// Verify optional fields are empty/nil
	assert.Len(t, config.Workflow.ConfigMaps, 0)
	assert.Len(t, config.Workflow.Secrets, 0)
	assert.Len(t, config.Workflow.Env, 0)
}

func TestLoadConfig_WithoutWorkflowConfig(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 3
baseBranch: main
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	// Verify workflow config exists but is empty
	assert.Equal(t, "", config.Workflow.Image.Repository)
	assert.Equal(t, "", config.Workflow.Image.Tag)
	assert.Equal(t, "", config.Workflow.Context)
	assert.Equal(t, "", config.Workflow.Namespace)
}

func TestApplyDefaults_Model(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `model: ""
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, "deepseek/deepseek-chat", config.Model)
}

func TestApplyDefaults_AppFields(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `app:
  name: ""
  id: ""
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, "zalphen", config.App.Name)
	assert.Equal(t, "2924254", config.App.ID)
}

func TestApplyDefaults_ServiceTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `services:
  - name: svc1
    command: echo
    timeout: 0
  - name: svc2
    command: echo
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	require.Len(t, config.Services, 2)
	assert.Equal(t, 30, config.Services[0].Timeout)
	assert.Equal(t, 30, config.Services[1].Timeout)
}

func TestApplyDefaults_DoesNotOverwriteNonZeroValues(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 5
baseBranch: develop
model: anthropic/claude-3-sonnet
app:
  name: my-app
  id: 1234567
services:
  - name: svc1
    command: echo
    timeout: 60
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, 5, config.MaxIterations)
	assert.Equal(t, "develop", config.BaseBranch)
	assert.Equal(t, "anthropic/claude-3-sonnet", config.Model)
	assert.Equal(t, "my-app", config.App.Name)
	assert.Equal(t, "1234567", config.App.ID)
	assert.Equal(t, 60, config.Services[0].Timeout)
}

func TestLoadConfig_CommentInstructionsFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 3
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	customCommentInstructions := "Custom comment instructions for PR comments"
	commentInstructionsPath := filepath.Join(ralphDir, "comment-instructions.md")
	require.NoError(t, os.WriteFile(commentInstructionsPath, []byte(customCommentInstructions), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, customCommentInstructions, config.CommentInstructions)
}

func TestLoadConfig_MergeInstructionsFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 3
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	customMergeInstructions := "Custom merge instructions for PR merging"
	mergeInstructionsPath := filepath.Join(ralphDir, "merge-instructions.md")
	require.NoError(t, os.WriteFile(mergeInstructionsPath, []byte(customMergeInstructions), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.Equal(t, customMergeInstructions, config.MergeInstructions)
}

func TestLoadConfig_DefaultInstructionsWhenFilesNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `maxIterations: 3
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")

	assert.NotEmpty(t, config.CommentInstructions, "CommentInstructions is empty, expected default instructions")
	assert.NotEmpty(t, config.MergeInstructions, "MergeInstructions is empty, expected default instructions")
	assert.NotEmpty(t, config.Instructions, "Instructions is empty, expected default instructions")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	invalidYAML := `maxIterations: [this is not valid yaml
  - broken
 `
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(invalidYAML), 0644))

	t.Chdir(tmpDir)

	_, err := LoadConfig()
	require.Error(t, err, "LoadConfig() expected error for invalid YAML")
}

func TestLoadProject_FileNotFound(t *testing.T) {
	_, err := LoadProject("/nonexistent/path/project.yaml")
	require.Error(t, err, "LoadProject() expected error for nonexistent file")
}

func TestSaveProjectRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	project := &Project{
		Name:        "round-trip-test",
		Description: "Testing save and load round trip",
		Requirements: []Requirement{
			{ID: "req1", Name: "Requirement 1", Passing: true},
			{ID: "req2", Name: "Requirement 2", Passing: false},
		},
	}

	projectPath := filepath.Join(tmpDir, "roundtrip.yaml")
	require.NoError(t, SaveProject(projectPath, project), "SaveProject() unexpected error")

	loaded, err := LoadProject(projectPath)
	require.NoError(t, err, "LoadProject() after save unexpected error")
	assert.Equal(t, project.Name, loaded.Name)
	assert.Equal(t, project.Description, loaded.Description)
	assert.Len(t, loaded.Requirements, len(project.Requirements))
}

func TestFindConfigDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ralph directory
	ralphDir := filepath.Join(tmpDir, ".ralph")
	err := os.MkdirAll(ralphDir, 0755)
	require.NoError(t, err)

	// Test finding from the same directory
	found, err := FindConfigDir(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, ralphDir, found)

	// Test finding from a subdirectory
	subDir := filepath.Join(tmpDir, "a", "b", "c")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	found, err = FindConfigDir(subDir)
	require.NoError(t, err)
	assert.Equal(t, ralphDir, found)

	// Test not finding it
	otherDir := t.TempDir()
	_, err = FindConfigDir(otherDir)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
