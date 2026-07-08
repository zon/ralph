package project

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func FileWithRequirement(t *testing.T, slug string, passing bool) string {
	t.Helper()

	proj := Project{
		Slug: "test-project",
		Requirements: []Requirement{
			{
				Slug:        slug,
				Description: "test requirement",
				Items:       []string{"test item"},
				Passing:     passing,
			},
		},
	}

	data, err := yaml.Marshal(proj)
	if err != nil {
		t.Fatalf("failed to marshal project: %v", err)
	}

	path := filepath.Join(t.TempDir(), "project.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write project file: %v", err)
	}

	return path
}

func RequirementStatus(t *testing.T, path, slug string) bool {
	t.Helper()

	proj, err := LoadProject(path)
	if err != nil {
		t.Fatalf("failed to load project from %s: %v", path, err)
	}

	for _, req := range proj.Requirements {
		if req.Slug == slug {
			return req.Passing
		}
	}

	t.Fatalf("requirement %q not found in project at %s", slug, path)
	return false
}

func NonExistentFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "nonexistent.yaml")
}

func Any() *Project {
	return &Project{
		Slug:  "test-project",
		Title: "Test Project",
		Requirements: []Requirement{
			{
				Slug:        "req-1",
				Description: "Requirement 1",
				Items:       []string{"Item 1"},
				Passing:     false,
			},
		},
	}
}

func WithAllPassing() *Project {
	return &Project{
		Slug:  "test-project",
		Title: "Test Project",
		Requirements: []Requirement{
			{
				Slug:        "req-1",
				Description: "Requirement 1",
				Items:       []string{"Item 1"},
				Passing:     true,
			},
		},
	}
}

func WithFailingRequirements() *Project {
	return &Project{
		Slug:  "test-project",
		Title: "Test Project",
		Requirements: []Requirement{
			{
				Slug:        "req-1",
				Description: "Requirement 1",
				Items:       []string{"Item 1"},
				Passing:     false,
			},
		},
	}
}

func WithFailingRequirementsCount(n int) *Project {
	reqs := make([]Requirement, n)
	for i := 0; i < n; i++ {
		reqs[i] = Requirement{
			Slug:        fmt.Sprintf("req-%d", i+1),
			Description: fmt.Sprintf("Requirement %d", i+1),
			Items:       []string{fmt.Sprintf("Item %d", i+1)},
			Passing:     false,
		}
	}
	return &Project{
		Slug:         "test-project",
		Title:        "Test Project",
		Requirements: reqs,
	}
}

func ForProjectInput(p *Project) *InputFile {
	return &InputFile{
		path:    p.Path,
		kind:    inputProject,
		project: p,
	}
}

func ForOrchestrationInput(path string) *InputFile {
	return &InputFile{
		path: path,
		kind: inputOrchestration,
	}
}

func ForSpecInput(path string) *InputFile {
	return &InputFile{
		path: path,
		kind: inputSpec,
	}
}

var anyPathValue = "/workspace/repo/projects/test-project.yaml"

func AnyPath() string {
	return anyPathValue
}

var anyJSONPathValue = "/workspace/repo/projects/test-project.json"

func AnyJSONPath() string {
	return anyJSONPathValue
}

var lastSavedValue *Project

func LastSaved() *Project {
	return lastSavedValue
}

func SetLastSaved(p *Project) {
	lastSavedValue = p
}

