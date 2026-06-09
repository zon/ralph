package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeProjectFile(t *testing.T, dir, name, slug string, passing ...bool) string {
	t.Helper()
	if len(passing) == 0 {
		passing = []bool{true}
	}
	reqs := make([]Requirement, len(passing))
	for i, p := range passing {
		reqs[i] = Requirement{
			Slug:        slug + "-req-" + string(rune('0'+i)),
			Description: "requirement",
			Items:       []string{"item"},
			Passing:     p,
		}
	}
	proj := Project{
		Slug:         slug,
		Title:        "Test " + slug,
		Requirements: reqs,
	}
	data, err := yaml.Marshal(proj)
	require.NoError(t, err)
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}

func TestLoadAll(t *testing.T) {
	t.Run("loads all yaml and yml files", func(t *testing.T) {
		dir := t.TempDir()
		writeProjectFile(t, dir, "proj-a.yaml", "proj-a", true)
		writeProjectFile(t, dir, "proj-b.yml", "proj-b", false)

		projects, err := LoadAll(dir)
		require.NoError(t, err)
		assert.Len(t, projects, 2)
	})

	t.Run("skips files that fail to load", func(t *testing.T) {
		dir := t.TempDir()
		writeProjectFile(t, dir, "good.yaml", "good", true)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("invalid: yaml: [[["), 0644))

		projects, err := LoadAll(dir)
		require.NoError(t, err)
		assert.Len(t, projects, 1)
		assert.Equal(t, "good", projects[0].Slug)
	})

	t.Run("skips non-yaml files", func(t *testing.T) {
		dir := t.TempDir()
		writeProjectFile(t, dir, "proj.yaml", "proj", true)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644))

		projects, err := LoadAll(dir)
		require.NoError(t, err)
		assert.Len(t, projects, 1)
	})

	t.Run("returns nil nil for non-existent directory", func(t *testing.T) {
		projects, err := LoadAll("/nonexistent/path/xyz123")
		assert.NoError(t, err)
		assert.Nil(t, projects)
	})

	t.Run("returns empty slice for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		projects, err := LoadAll(dir)
		require.NoError(t, err)
		assert.Empty(t, projects)
	})
}

func TestFilterPassing(t *testing.T) {
	makeProj := func(slug string, passing bool) *Project {
		return &Project{
			Slug: slug,
			Requirements: []Requirement{
				{Slug: slug + "-req", Items: []string{"item"}, Passing: passing},
			},
		}
	}

	t.Run("filters to only passing projects", func(t *testing.T) {
		projects := []*Project{
			makeProj("passing-1", true),
			makeProj("failing-1", false),
			makeProj("passing-2", true),
		}
		result := FilterPassing(projects)
		assert.Len(t, result, 2)
		assert.Equal(t, "passing-1", result[0].Slug)
		assert.Equal(t, "passing-2", result[1].Slug)
	})

	t.Run("returns empty slice when none passing", func(t *testing.T) {
		projects := []*Project{
			makeProj("failing-1", false),
			makeProj("failing-2", false),
		}
		result := FilterPassing(projects)
		assert.Empty(t, result)
	})

	t.Run("returns all when all passing", func(t *testing.T) {
		projects := []*Project{
			makeProj("passing-1", true),
			makeProj("passing-2", true),
		}
		result := FilterPassing(projects)
		assert.Len(t, result, 2)
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		result := FilterPassing(nil)
		assert.Nil(t, result)
	})
}

func TestDeleteAll(t *testing.T) {
	t.Run("deletes all project files", func(t *testing.T) {
		dir := t.TempDir()
		p1 := writeProjectFile(t, dir, "a.yaml", "proj-a", true)
		p2 := writeProjectFile(t, dir, "b.yaml", "proj-b", false)

		projects := []*Project{
			{Path: p1},
			{Path: p2},
		}

		err := DeleteAll(projects)
		require.NoError(t, err)

		_, err = os.Stat(p1)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(p2)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("errors on non-existent file", func(t *testing.T) {
		projects := []*Project{
			{Path: "/nonexistent/path/file.yaml"},
		}
		err := DeleteAll(projects)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete")
	})

	t.Run("no error for empty slice", func(t *testing.T) {
		err := DeleteAll(nil)
		assert.NoError(t, err)
	})
}
