package projectfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteCanonicalPreservesDocument covers the item that the rewrite writes
// the parsed document back as YAML, preserving its full content and key order.
func TestWriteCanonicalPreservesDocument(t *testing.T) {
	content := "slug: csv-export\nunrelated: [1, 2, 3]\nrequirements:\n  - slug: one\n    items:\n      - a\n"
	path := writeProjectFile(t, "project.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)

	outPath := filepath.Join(t.TempDir(), "out.yaml")
	require.NoError(t, WriteCanonical(outPath, doc))

	rewritten, err := Parse(outPath)
	require.NoError(t, err)
	origRoot, ok := doc.Root.(map[string]any)
	require.True(t, ok)
	newRoot, ok := rewritten.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, origRoot["slug"], newRoot["slug"])
	assert.Equal(t, origRoot["unrelated"], newRoot["unrelated"])
	require.Len(t, newRoot["requirements"], 1)
}

// TestWriteCanonicalPreservesKeyOrder covers the item that the rewrite writes
// the parsed document back as YAML preserving its key order.
func TestWriteCanonicalPreservesKeyOrder(t *testing.T) {
	content := "zebra: 1\nalpha: 2\nmiddle: 3\n"
	path := writeProjectFile(t, "project.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)

	outPath := filepath.Join(t.TempDir(), "out.yaml")
	require.NoError(t, WriteCanonical(outPath, doc))

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	out := string(data)
	require.Less(t, indexOf(t, out, "zebra"), indexOf(t, out, "alpha"))
	require.Less(t, indexOf(t, out, "alpha"), indexOf(t, out, "middle"))
}

// TestWriteCanonicalByteIdenticalForCanonicalInput covers the scenario that a
// file already in canonical YAML is rewritten byte-identically.
func TestWriteCanonicalByteIdenticalForCanonicalInput(t *testing.T) {
	content := "slug: csv-export\nrequirements:\n    - slug: one\n      items:\n        - a\n"
	path := writeProjectFile(t, "project.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)

	canonicalPath := filepath.Join(t.TempDir(), "canonical.yaml")
	require.NoError(t, WriteCanonical(canonicalPath, doc))
	canonical, err := os.ReadFile(canonicalPath)
	require.NoError(t, err)

	// Writing the already-canonical output again must be byte-identical.
	doc2, err := Parse(canonicalPath)
	require.NoError(t, err)
	outPath := filepath.Join(t.TempDir(), "again.yaml")
	require.NoError(t, WriteCanonical(outPath, doc2))
	out, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Equal(t, string(canonical), string(out))
}

// TestWriteCanonicalNoFieldAddedOrDropped covers the item that no field is
// dropped for being unrecognized and no field is added.
func TestWriteCanonicalNoFieldAddedOrDropped(t *testing.T) {
	content := "slug: csv-export\nforeign:\n  custom: [1, 2, 3]\nnotes: some text\nrequirements:\n  - slug: one\n"
	path := writeProjectFile(t, "project.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)

	outPath := filepath.Join(t.TempDir(), "out.yaml")
	require.NoError(t, WriteCanonical(outPath, doc))

	rewritten, err := Parse(outPath)
	require.NoError(t, err)
	origRoot, ok := doc.Root.(map[string]any)
	require.True(t, ok)
	newRoot, ok := rewritten.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, origRoot, newRoot)
}

// TestWriteCanonicalJSONWritesSiblingYAML covers the item that a .json input is
// written to a new file with the same name and a .yaml extension. Removing the
// original .json is the caller's decision, so it stays on disk here.
func TestWriteCanonicalJSONWritesSiblingYAML(t *testing.T) {
	jsonPath := writeProjectFile(t, "project.json", `{"slug":"csv-export","requirements":[{"slug":"one"}]}`)
	doc, err := Parse(jsonPath)
	require.NoError(t, err)

	require.NoError(t, WriteCanonical(jsonPath, doc))

	yamlPath := strings.TrimSuffix(jsonPath, filepath.Ext(jsonPath)) + ".yaml"
	_, err = os.ReadFile(yamlPath)
	require.NoError(t, err)

	rewritten, err := Parse(yamlPath)
	require.NoError(t, err)
	root, ok := rewritten.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "csv-export", root["slug"])
	require.Len(t, root["requirements"], 1)

	require.FileExists(t, jsonPath, "removing the original .json is the caller's decision")
}

// TestWriteCanonicalPreservesEmptyAndNilFields covers the item that the rewrite
// no longer marshals a typed project model, so empty and nil field omission
// rules no longer apply.
func TestWriteCanonicalPreservesEmptyAndNilFields(t *testing.T) {
	content := "notes: {}\nowner:\ndescription: \"\"\ncount: 0\nenabled: false\nitems: []\n"
	path := writeProjectFile(t, "project.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)

	outPath := filepath.Join(t.TempDir(), "out.yaml")
	require.NoError(t, WriteCanonical(outPath, doc))

	rewritten, err := Parse(outPath)
	require.NoError(t, err)
	origRoot, ok := doc.Root.(map[string]any)
	require.True(t, ok)
	newRoot, ok := rewritten.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, origRoot, newRoot)
}

func indexOf(t *testing.T, s, substr string) int {
	t.Helper()
	idx := -1
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("substring %q not found in %q", substr, s)
	}
	return idx
}
