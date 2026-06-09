package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputFilePredicates(t *testing.T) {
	t.Run("IsProject returns true for project kind", func(t *testing.T) {
		f := &InputFile{kind: inputProject}
		assert.True(t, f.IsProject())
		assert.False(t, f.IsSpec())
		assert.False(t, f.IsOrchestration())
	})

	t.Run("IsSpec returns true for spec kind", func(t *testing.T) {
		f := &InputFile{kind: inputSpec}
		assert.True(t, f.IsSpec())
		assert.False(t, f.IsProject())
		assert.False(t, f.IsOrchestration())
	})

	t.Run("IsOrchestration returns true for orchestration kind", func(t *testing.T) {
		f := &InputFile{kind: inputOrchestration}
		assert.True(t, f.IsOrchestration())
		assert.False(t, f.IsProject())
		assert.False(t, f.IsSpec())
	})

	t.Run("Path returns the stored path", func(t *testing.T) {
		f := &InputFile{path: "/some/path.yaml"}
		assert.Equal(t, "/some/path.yaml", f.Path())
	})

	t.Run("Project returns the stored project", func(t *testing.T) {
		p := &Project{Slug: "my-project"}
		f := &InputFile{kind: inputProject, project: p}
		assert.Equal(t, p, f.Project())
	})
}

func TestInputFileSlug(t *testing.T) {
	t.Run("project input returns project slug", func(t *testing.T) {
		f := &InputFile{
			kind:    inputProject,
			project: &Project{Slug: "my-feature"},
		}
		assert.Equal(t, "my-feature", f.Slug())
	})

	t.Run("orchestration input derives slug from parent directory", func(t *testing.T) {
		f := &InputFile{
			path: "/workspace/repo/specs/features/my-feature/orchestration.md",
			kind: inputOrchestration,
		}
		assert.Equal(t, "my-feature", f.Slug())
	})

	t.Run("orchestration input sanitizes directory name with spaces", func(t *testing.T) {
		f := &InputFile{
			path: "/tmp/My Feature/orchestration.md",
			kind: inputOrchestration,
		}
		assert.Equal(t, "my-feature", f.Slug())
	})

	t.Run("spec input derives slug from parent directory", func(t *testing.T) {
		f := &InputFile{
			path: "/tmp/specs/features/my-feature/spec.md",
			kind: inputSpec,
		}
		assert.Equal(t, "my-feature", f.Slug())
	})
}

func TestResolveInputFile(t *testing.T) {
	t.Run("loads project from .yaml file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "project.yaml")
		require.NoError(t, os.WriteFile(path, []byte("slug: my-project\nrequirements:\n  - slug: req-1\n    items:\n      - item 1\n    passing: false\n"), 0644))

		f, err := ResolveInputFile(path)
		require.NoError(t, err)
		assert.True(t, f.IsProject())
		assert.Equal(t, "my-project", f.Slug())
		assert.NotNil(t, f.Project())
	})

	t.Run("loads project from .yml file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "project.yml")
		require.NoError(t, os.WriteFile(path, []byte("slug: my-project\nrequirements:\n  - slug: req-1\n    items:\n      - item 1\n    passing: false\n"), 0644))

		f, err := ResolveInputFile(path)
		require.NoError(t, err)
		assert.True(t, f.IsProject())
		assert.Equal(t, "my-project", f.Slug())
	})

	t.Run("detects orchestration.md file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "orchestration.md")
		require.NoError(t, os.WriteFile(path, []byte("# Orchestration\n"), 0644))

		f, err := ResolveInputFile(path)
		require.NoError(t, err)
		assert.True(t, f.IsOrchestration())
		assert.Equal(t, filepath.Base(dir), f.Slug())
	})

	t.Run("detects spec.md file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "spec.md")
		require.NoError(t, os.WriteFile(path, []byte("# Spec\n"), 0644))

		f, err := ResolveInputFile(path)
		require.NoError(t, err)
		assert.True(t, f.IsSpec())
		assert.Equal(t, filepath.Base(dir), f.Slug())
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nonexistent.yaml")
		_, err := ResolveInputFile(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input file not found")
	})

	t.Run("returns error for unrecognized file type", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "readme.txt")
		require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))

		_, err := ResolveInputFile(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unrecognized input file type")
	})
}

func TestResolveInputFile_ProjectSlugFromYAML(t *testing.T) {
	t.Run("slug comes from project YAML slug field", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "project.yaml")
		require.NoError(t, os.WriteFile(path, []byte("slug: from-yaml-field\nrequirements:\n  - slug: req-1\n    items:\n      - item 1\n    passing: false\n"), 0644))

		f, err := ResolveInputFile(path)
		require.NoError(t, err)
		assert.Equal(t, "from-yaml-field", f.Slug())
	})
}

func TestResolveInputFile_InvalidProject(t *testing.T) {
	t.Run("returns error for invalid project YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.yaml")
		require.NoError(t, os.WriteFile(path, []byte("slug: \nrequirements: []\n"), 0644))

		_, err := ResolveInputFile(path)
		require.Error(t, err)
	})
}

func TestTestHelpers(t *testing.T) {
	t.Run("ForProjectInput creates InputFile with project kind", func(t *testing.T) {
		p := &Project{Slug: "test-project", Path: "/tmp/test.yaml"}
		f := ForProjectInput(p)
		assert.Equal(t, p.Path, f.Path())
		assert.True(t, f.IsProject())
		assert.Equal(t, p, f.Project())
	})

	t.Run("ForOrchestrationInput creates InputFile with orchestration kind", func(t *testing.T) {
		f := ForOrchestrationInput("/tmp/orchestration.md")
		assert.True(t, f.IsOrchestration())
		assert.Equal(t, "/tmp/orchestration.md", f.Path())
		assert.Nil(t, f.Project())
	})

	t.Run("ForSpecInput creates InputFile with spec kind", func(t *testing.T) {
		f := ForSpecInput("/tmp/spec.md")
		assert.True(t, f.IsSpec())
		assert.Equal(t, "/tmp/spec.md", f.Path())
		assert.Nil(t, f.Project())
	})
}
