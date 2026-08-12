package projectfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeProjectFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestParseYAMLByExtension(t *testing.T) {
	path := writeProjectFile(t, "project.yaml", "slug: csv-export\nrequirements:\n  - slug: one\n  - slug: two\n")
	doc, err := Parse(path)
	require.NoError(t, err)
	root, ok := doc.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "csv-export", root["slug"])
}

func TestParseYMLByExtension(t *testing.T) {
	path := writeProjectFile(t, "project.yml", "- one\n- two\n")
	doc, err := Parse(path)
	require.NoError(t, err)
	items, ok := doc.Root.([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"one", "two"}, items)
}

func TestParseJSONByExtension(t *testing.T) {
	path := writeProjectFile(t, "project.json", `{"requirements": ["one", "two"]}`)
	doc, err := Parse(path)
	require.NoError(t, err)
	root, ok := doc.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"one", "two"}, root["requirements"])
}

func TestParseRejectsUnsupportedExtension(t *testing.T) {
	path := writeProjectFile(t, "project.md", "# not a project\n")
	_, err := Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized input file type")
}

func TestParseMissingFileReturnsError(t *testing.T) {
	_, err := Parse(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
}

func TestParsePreservesRawContent(t *testing.T) {
	content := "slug: csv-export\ntitle: CSV export\nrequirements:\n  - slug: one\n    description: first\n  - slug: two\n"
	path := writeProjectFile(t, "project.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)
	assert.Equal(t, content, doc.Raw)
}

func TestParseRetainsFieldsOutsideTheItemArray(t *testing.T) {
	content := "slug: csv-export\nunrelated: [1, 2, 3]\nrequirements:\n  - slug: one\n"
	path := writeProjectFile(t, "project.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)
	root, ok := doc.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{1, 2, 3}, root["unrelated"])
}
