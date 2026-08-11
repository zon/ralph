package projectfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDefaultQueryOnTopLevelArray(t *testing.T) {
	path := writeProjectFile(t, "project.yaml", "- Add a CSV serializer\n- Add an export endpoint\n- Return 404 for unknown reports\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	items, err := ResolveItems(doc, ".")
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, "Add a CSV serializer", items[0])
}

func TestResolveSingleArrayOutputUsesElements(t *testing.T) {
	path := writeProjectFile(t, "project.yaml", "requirements:\n  - slug: one\n  - slug: two\n  - slug: three\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	items, err := ResolveItems(doc, ".requirements")
	require.NoError(t, err)
	require.Len(t, items, 3)
	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "one", item["slug"])
}

func TestResolveArrayIterationMatchesArrayForm(t *testing.T) {
	path := writeProjectFile(t, "project.yaml", "requirements:\n  - slug: one\n  - slug: two\n  - slug: three\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	fromArray, err := ResolveItems(doc, ".requirements")
	require.NoError(t, err)
	fromIteration, err := ResolveItems(doc, ".requirements[]")
	require.NoError(t, err)
	assert.Equal(t, fromArray, fromIteration)
}

func TestResolveScalarOutputIsSingleItem(t *testing.T) {
	path := writeProjectFile(t, "project.yaml", "name: only-item\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	items, err := ResolveItems(doc, ".name")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "only-item", items[0])
}

func TestResolveMultipleOutputsAreItems(t *testing.T) {
	path := writeProjectFile(t, "project.yaml", "foo: 1\nbar: 2\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	items, err := ResolveItems(doc, ".foo, .bar")
	require.NoError(t, err)
	assert.Equal(t, []any{1, 2}, items)
}

func TestResolveNoOutputReturnsError(t *testing.T) {
	path := writeProjectFile(t, "project.yaml", "foo: 1\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	_, err = ResolveItems(doc, "empty")
	require.Error(t, err)
	assert.Equal(t, "item query yielded no items: empty", err.Error())
}

func TestResolveEvaluationErrorNamesQuery(t *testing.T) {
	path := writeProjectFile(t, "project.yaml", "foo: 1\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	_, err = ResolveItems(doc, ".foo.bar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "item query failed")
	assert.Contains(t, err.Error(), ".foo.bar")
	assert.NotContains(t, err.Error(), "yielded no items")
}

func TestResolveInvalidQueryNamesQuery(t *testing.T) {
	path := writeProjectFile(t, "project.yaml", "foo: 1\n")
	doc, err := Parse(path)
	require.NoError(t, err)

	_, err = ResolveItems(doc, ".[")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "item query failed")
	assert.NotContains(t, err.Error(), "yielded no items")
}

func TestResolveShapeIndependence(t *testing.T) {
	content := "- Add a CSV serializer\n- slug: export-endpoint\n  description: Expose the export over HTTP\n- nested:\n    deep:\n      value: 1\n"
	path := writeProjectFile(t, "project.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)

	items, err := ResolveItems(doc, ".")
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, "Add a CSV serializer", items[0])
	assert.IsType(t, map[string]any{}, items[1])
	assert.IsType(t, map[string]any{}, items[2])
}

func TestResolveForeignYAMLFileAsProject(t *testing.T) {
	content := "jobs:\n  - name: build\n    steps: [checkout, test]\n  - name: deploy\n    steps: [build, push]\n"
	path := writeProjectFile(t, "ci.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)

	items, err := ResolveItems(doc, ".jobs")
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestResolveForeignJSONFileAsProject(t *testing.T) {
	content := `{"tasks": [{"id": 1, "state": "open"}, {"id": 2, "state": "open"}]}`
	path := writeProjectFile(t, "issues.json", content)
	doc, err := Parse(path)
	require.NoError(t, err)

	items, err := ResolveItems(doc, ".tasks")
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestResolveEvaluatesJQExpression(t *testing.T) {
	content := "- slug: a\n  assignee: ralph\n- slug: b\n  assignee: eve\n"
	path := writeProjectFile(t, "project.yaml", content)
	doc, err := Parse(path)
	require.NoError(t, err)

	items, err := ResolveItems(doc, `. | map(select(.assignee == "ralph"))`)
	require.NoError(t, err)
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "a", item["slug"])
}
