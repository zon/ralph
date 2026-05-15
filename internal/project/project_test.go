package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validRequirement(slug string) Requirement {
	return Requirement{
		Slug:        slug,
		Description: "test requirement",
		Items:       []string{"Item 1"},
		Passing:     false,
	}
}

func TestValidateProject(t *testing.T) {
	tests := []struct {
		name    string
		project *Project
		wantErr bool
	}{
		{
			name: "valid project with items",
			project: &Project{
				Slug:         "test-project",
				Requirements: []Requirement{validRequirement("req-1")},
			},
			wantErr: false,
		},
		{
			name: "valid project with scenarios",
			project: &Project{
				Slug: "test-project",
				Requirements: []Requirement{
					{
						Slug:        "req-1",
						Description: "Scenario-only requirement",
						Scenarios: []Scenario{
							{Title: "Happy path", Items: []string{"GIVEN", "WHEN", "THEN"}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid project with code and tests",
			project: &Project{
				Slug: "test-project",
				Requirements: []Requirement{
					{
						Slug:        "req-1",
						Description: "Has code and tests",
						Code: []CodeEntry{
							{Name: "Foo", Description: "does foo", Module: "internal/foo", Body: "func Foo()"},
						},
						Tests: []CodeEntry{
							{Name: "TestFoo", Description: "tests foo", Module: "internal/foo", Body: "func TestFoo(t *testing.T)"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing slug",
			project: &Project{
				Slug:         "",
				Requirements: []Requirement{validRequirement("req-1")},
			},
			wantErr: true,
		},
		{
			name: "no requirements",
			project: &Project{
				Slug:         "test-project",
				Requirements: []Requirement{},
			},
			wantErr: true,
		},
		{
			name: "requirement missing slug",
			project: &Project{
				Slug: "test-project",
				Requirements: []Requirement{
					{Description: "no slug", Items: []string{"x"}},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate requirement slugs",
			project: &Project{
				Slug: "test-project",
				Requirements: []Requirement{
					validRequirement("req-1"),
					validRequirement("req-1"),
				},
			},
			wantErr: true,
		},
		{
			name: "requirement with no items/scenarios/code/tests",
			project: &Project{
				Slug: "test-project",
				Requirements: []Requirement{
					{Slug: "req-1", Description: "empty"},
				},
			},
			wantErr: true,
		},
		{
			name: "code entry missing fields",
			project: &Project{
				Slug: "test-project",
				Requirements: []Requirement{
					{
						Slug:        "req-1",
						Description: "broken code",
						Code:        []CodeEntry{{Name: "Foo"}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test entry missing fields",
			project: &Project{
				Slug: "test-project",
				Requirements: []Requirement{
					{
						Slug:        "req-1",
						Description: "broken test",
						Tests:       []CodeEntry{{Name: "TestFoo", Module: "internal/foo"}},
					},
				},
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
				Slug: "test",
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
				Slug: "test",
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
				Slug: "test",
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
		Slug: "test",
		Requirements: []Requirement{
			{Slug: "req-1", Passing: false},
			{Slug: "req-2", Passing: false},
		},
	}

	require.NoError(t, UpdateRequirementStatus(proj, "req-1", true))
	assert.True(t, proj.Requirements[0].Passing, "expected req-1 to be marked passing")
	assert.False(t, proj.Requirements[1].Passing, "req-2 should be untouched")

	require.Error(t, UpdateRequirementStatus(proj, "req-999", true), "expected error for unknown slug")
}

func TestLoadProject(t *testing.T) {
	tmpDir := t.TempDir()

	projectContent := `slug: test-project
title: A test project
requirements:
  - slug: first-requirement
    description: First requirement
    items:
      - Item A
    passing: false
  - slug: second-requirement
    description: Second requirement
    items:
      - Item B
    passing: true
`
	projectPath := filepath.Join(tmpDir, "test-project.yaml")
	require.NoError(t, os.WriteFile(projectPath, []byte(projectContent), 0644))

	proj, err := LoadProject(projectPath)
	require.NoError(t, err)

	assert.Equal(t, "test-project", proj.Slug)
	assert.Equal(t, "A test project", proj.Title)
	require.Len(t, proj.Requirements, 2)
	assert.Equal(t, "first-requirement", proj.Requirements[0].Slug)
}

func TestLoadProjectWithCodeAndTests(t *testing.T) {
	tmpDir := t.TempDir()

	projectContent := `slug: full-shape
title: Project exercising all requirement fields
feature: specs/features/foo/bar
requirements:
  - slug: implement-feature
    description: Implement feature X
    scenarios:
      - title: Happy path
        items:
          - GIVEN a user
          - WHEN they do the thing
          - THEN it works
    code:
      - name: DoThing
        description: performs the thing
        module: internal/thing
        body: |
          func DoThing() error
    tests:
      - name: TestDoThing
        description: verifies DoThing succeeds
        module: internal/thing
        body: |
          func TestDoThing(t *testing.T)
    passing: false
`
	projectPath := filepath.Join(tmpDir, "full-shape.yaml")
	require.NoError(t, os.WriteFile(projectPath, []byte(projectContent), 0644))

	proj, err := LoadProject(projectPath)
	require.NoError(t, err)

	assert.Equal(t, "full-shape", proj.Slug)
	assert.Equal(t, "specs/features/foo/bar", proj.Feature)
	require.Len(t, proj.Requirements, 1)
	req := proj.Requirements[0]
	assert.Equal(t, "implement-feature", req.Slug)
	require.Len(t, req.Scenarios, 1)
	assert.Equal(t, "Happy path", req.Scenarios[0].Title)
	require.Len(t, req.Code, 1)
	assert.Equal(t, "DoThing", req.Code[0].Name)
	assert.Equal(t, "internal/thing", req.Code[0].Module)
	require.Len(t, req.Tests, 1)
	assert.Equal(t, "TestDoThing", req.Tests[0].Name)
}

func TestSaveProjectRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	proj := &Project{
		Slug:  "round-trip",
		Title: "Testing save and load round trip",
		Requirements: []Requirement{
			{Slug: "req-1", Description: "Req 1", Items: []string{"a"}, Passing: true},
			{Slug: "req-2", Description: "Req 2", Items: []string{"b"}, Passing: false},
		},
	}

	projectPath := filepath.Join(tmpDir, "roundtrip.yaml")
	require.NoError(t, SaveProject(projectPath, proj))

	loaded, err := LoadProject(projectPath)
	require.NoError(t, err)
	assert.Equal(t, proj.Slug, loaded.Slug)
	assert.Equal(t, proj.Title, loaded.Title)
	require.Len(t, loaded.Requirements, 2)
	assert.Equal(t, "req-1", loaded.Requirements[0].Slug)
}

func TestLoadProject_FileNotFound(t *testing.T) {
	_, err := LoadProject("/nonexistent/path/project.yaml")
	require.Error(t, err)
}
