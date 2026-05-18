package project

import (
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