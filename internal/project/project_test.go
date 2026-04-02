package project

import (
	"os"
	"path/filepath"
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
	proj := &Project{
		Name: "test",
		Requirements: []Requirement{
			{ID: "req1", Passing: false},
			{ID: "req2", Passing: false},
		},
	}

	// Update existing requirement
	err := UpdateRequirementStatus(proj, "req1", true)
	require.NoError(t, err, "UpdateRequirementStatus() unexpected error")
	assert.True(t, proj.Requirements[0].Passing, "UpdateRequirementStatus() did not update status")

	// Try to update non-existent requirement
	err = UpdateRequirementStatus(proj, "req999", true)
	require.Error(t, err, "UpdateRequirementStatus() expected error for non-existent requirement")
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

func TestSaveProject(t *testing.T) {
	tmpDir := t.TempDir()

	proj := &Project{
		Name:        "test-project",
		Description: "Test description",
		Requirements: []Requirement{
			{ID: "req1", Name: "Requirement 1", Passing: true},
		},
	}

	projectPath := filepath.Join(tmpDir, "project.yaml")
	require.NoError(t, SaveProject(projectPath, proj), "SaveProject() unexpected error")

	// Verify file was created
	_, err := os.Stat(projectPath)
	require.NoError(t, err, "SaveProject() did not create file")

	// Load it back and verify
	loaded, err := LoadProject(projectPath)
	require.NoError(t, err, "LoadProject() after save unexpected error")
	assert.Equal(t, proj.Name, loaded.Name)
}

func TestLoadProject_FileNotFound(t *testing.T) {
	_, err := LoadProject("/nonexistent/path/project.yaml")
	require.Error(t, err, "LoadProject() expected error for nonexistent file")
}

func TestSaveProjectRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	proj := &Project{
		Name:        "round-trip-test",
		Description: "Testing save and load round trip",
		Requirements: []Requirement{
			{ID: "req1", Name: "Requirement 1", Passing: true},
			{ID: "req2", Name: "Requirement 2", Passing: false},
		},
	}

	projectPath := filepath.Join(tmpDir, "roundtrip.yaml")
	require.NoError(t, SaveProject(projectPath, proj), "SaveProject() unexpected error")

	loaded, err := LoadProject(projectPath)
	require.NoError(t, err, "LoadProject() after save unexpected error")
	assert.Equal(t, proj.Name, loaded.Name)
	assert.Equal(t, proj.Description, loaded.Description)
	assert.Len(t, loaded.Requirements, len(proj.Requirements))
}
