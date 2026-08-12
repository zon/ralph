package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeNamedProjectFile writes content to a file whose base name is name, under
// a `projects` subdirectory of a temp dir so paths mirror the scenarios.
func writeNamedProjectFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "projects")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestScenarioSlugFieldPresent(t *testing.T) {
	path := writeNamedProjectFile(t, "anything.yaml", "slug: csv-export\ntitle: CSV Export\nrequirements:\n  - slug: one\n  - slug: two\n")

	proj, err := (&Client{}).Resolve(path, ".requirements")
	require.NoError(t, err)
	assert.Equal(t, "csv-export", proj.Slug)
}

func TestScenarioTopLevelArrayHasNoSlugField(t *testing.T) {
	path := writeNamedProjectFile(t, "csv-export.yaml", "- one\n- two\n")

	proj, err := (&Client{}).Resolve(path, ".")
	require.NoError(t, err)
	assert.Equal(t, "csv-export", proj.Slug, "slug falls back to the file's base name")
}

func TestScenarioMappingWithoutSlugField(t *testing.T) {
	path := writeNamedProjectFile(t, "tasks.json", `{"tasks": ["one", "two"]}`)

	proj, err := (&Client{}).Resolve(path, ".tasks")
	require.NoError(t, err)
	assert.Equal(t, "tasks", proj.Slug, "slug falls back to the file's base name without its extension")
}

func TestScenarioProjectFileProvided(t *testing.T) {
	path := writeNamedProjectFile(t, "csv-export.yaml", "- one\n- two\n")

	input, err := ResolveInputFile(path)
	require.NoError(t, err)
	assert.True(t, input.IsProject())
	assert.Equal(t, "csv-export", input.Slug())
	require.NotNil(t, input.Project())
	require.Len(t, input.Project().Items, 2)
}

func TestScenarioUnrecognizedFileType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readme.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o644))

	absPath, err := filepath.Abs(path)
	require.NoError(t, err)
	_, err = ResolveInputFile(path)
	require.Error(t, err)
	assert.Equal(t, "unrecognized input file type: "+absPath, err.Error())
}

func TestScenarioTopLevelArrayHasNoMetadata(t *testing.T) {
	path := writeNamedProjectFile(t, "csv-export.yaml", "- one\n- two\n")

	proj, err := (&Client{}).Resolve(path, ".")
	require.NoError(t, err)
	assert.Equal(t, "csv-export", proj.Slug, "slug derived from the file name")
	assert.Equal(t, "csv-export", proj.Title, "title falls back to the resolved slug")
}
