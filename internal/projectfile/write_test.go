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
// preserves empty maps, nil values, zero numbers, and empty sequences exactly as
// the source document wrote them.
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

// TestWriteCanonicalReplacesExistingFileAtomically covers the item that
// writing over an existing project file replaces it without leaving temporary
// files behind.
func TestWriteCanonicalReplacesExistingFileAtomically(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "project.yaml")
	require.NoError(t, os.WriteFile(target, []byte("old: value\n"), 0o644))

	path := writeProjectFile(t, "project.yaml", "slug: csv-export\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	require.NoError(t, WriteCanonical(target, doc))

	rewritten, err := Parse(target)
	require.NoError(t, err)
	root, ok := rewritten.Root.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "csv-export", root["slug"])

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "project.yaml", entries[0].Name())
}

// TestWriteCanonicalLeavesTargetOnWriteFailure covers the item that a failed
// write cannot truncate or remove an existing target.
func TestWriteCanonicalLeavesTargetOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "project.yaml")
	require.NoError(t, os.Mkdir(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "keep"), []byte("keep"), 0o644))

	path := writeProjectFile(t, "project.yaml", "slug: csv-export\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	require.Error(t, WriteCanonical(target, doc))

	// The existing target is untouched, and no temporary file is left behind.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "project.yaml", entries[0].Name())
	data, err := os.ReadFile(filepath.Join(target, "keep"))
	require.NoError(t, err)
	assert.Equal(t, "keep", string(data))
}

// TestWriteCanonicalPreservesSourceMode covers the item that a rewritten file
// keeps the source file's permissions, including a private .json source whose
// canonical .yaml sibling is created new.
func TestWriteCanonicalPreservesSourceMode(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		content string
	}{
		{name: "yaml in place", file: "project.yaml", content: "slug: csv-export\n"},
		{name: "json sibling yaml", file: "project.json", content: `{"slug":"csv-export"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := writeProjectFile(t, tt.file, tt.content)
			require.NoError(t, os.Chmod(source, 0o600))
			doc, err := Parse(source)
			require.NoError(t, err)

			require.NoError(t, WriteCanonical(source, doc))

			target := CanonicalPath(source)
			info, err := os.Stat(target)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		})
	}
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
