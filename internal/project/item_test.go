package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/trailer"
)

func TestNewItemsAssignsZeroBasedIndices(t *testing.T) {
	items := NewItems([]any{"a", "b", "c", "d"})
	require.Len(t, items, 4)
	for i, item := range items {
		assert.Equal(t, i, item.Index)
	}
}

func TestItemCarriesRawValueUnchanged(t *testing.T) {
	value := map[string]any{
		"slug":    "export-endpoint",
		"nested":  []any{"x", map[string]any{"deep": 1}},
		"details": "unchanged",
	}
	items := NewItems([]any{value, "plain string"})

	assert.Equal(t, value, items[0].Value)
	assert.Equal(t, "plain string", items[1].Value)
}

func TestItemKeyFromSlug(t *testing.T) {
	item := NewItems([]any{map[string]any{"slug": "csv-serializer"}})[0]
	assert.Equal(t, "csv-serializer", item.Key())
}

func TestItemKeyFallsBackToID(t *testing.T) {
	item := NewItems([]any{map[string]any{"id": 4821}})[0]
	assert.Equal(t, "4821", item.Key())
}

func TestItemKeyFallsBackToName(t *testing.T) {
	item := NewItems([]any{map[string]any{"name": "export-endpoint"}})[0]
	assert.Equal(t, "export-endpoint", item.Key())
}

func TestItemKeyPrefersSlugOverIDAndName(t *testing.T) {
	item := NewItems([]any{
		map[string]any{"slug": "csv-serializer", "id": 4821, "name": "serializer"},
	})[0]
	assert.Equal(t, "csv-serializer", item.Key())
}

func TestItemKeyPrefersIDOverName(t *testing.T) {
	item := NewItems([]any{
		map[string]any{"id": 4821, "name": "export-endpoint"},
	})[0]
	assert.Equal(t, "4821", item.Key())
}

func TestItemWithNoKey(t *testing.T) {
	items := NewItems([]any{
		"plain string",
		map[string]any{"description": "no label fields"},
		map[string]any{"slug": []any{"not", "scalar"}},
		map[string]any{"name": nil},
	})
	for _, item := range items {
		assert.Empty(t, item.Key())
	}
}

func TestItemKeyRenderedAsString(t *testing.T) {
	item := NewItems([]any{map[string]any{"id": 4821}})[0]
	assert.Equal(t, "4821", item.Key())
}

func TestDuplicateKeysAreNotAnError(t *testing.T) {
	items := NewItems([]any{
		map[string]any{"slug": "shared-key"},
		map[string]any{"slug": "shared-key"},
	})
	require.Len(t, items, 2)
	assert.Equal(t, "shared-key", items[0].Key())
	assert.Equal(t, "shared-key", items[1].Key())
	assert.NotEqual(t, items[0].Index, items[1].Index)
}

func TestScenarioHashIdentifiesItem(t *testing.T) {
	items := NewItems([]any{"one", "two", "three", "four"})

	selected := items[2]
	assert.Equal(t, 2, selected.Index)
	assert.Equal(t, "csv-export-"+selected.Hash(), trailer.Format("csv-export", selected.Hash()))
	assert.Equal(t, selected.Hash(), trailer.Hash("three"))
}

func TestScenarioKeyTakenFromSlug(t *testing.T) {
	item := NewItems([]any{map[string]any{"slug": "csv-serializer"}})[0]

	assert.Equal(t, "csv-serializer", item.Key())
	assert.Equal(t, "csv-export-"+item.Hash(), trailer.Format("csv-export", item.Hash()))
}

func TestScenarioKeyFallsBackToIDThenName(t *testing.T) {
	item := NewItems([]any{map[string]any{"id": 4821}})[0]

	assert.Equal(t, "4821", item.Key())
}

func TestScenarioItemWithNoKey(t *testing.T) {
	keyed := NewItems([]any{map[string]any{"slug": "csv-serializer"}})[0]
	plain := NewItems([]any{"Add a CSV serializer"})[0]

	assert.Empty(t, plain.Key())
	assert.Equal(t, "csv-export-"+plain.Hash(), trailer.Format("csv-export", plain.Hash()))
	assert.Equal(t, 0, plain.Index, "tracked exactly as any keyed item is")
	assert.Equal(t, keyed.Index, plain.Index)
}

func TestScenarioIdenticalTextSharesHash(t *testing.T) {
	items := NewItems([]any{
		map[string]any{"slug": "duplicate"},
		map[string]any{"slug": "duplicate"},
	})
	require.Len(t, items, 2)
	assert.Equal(t, items[0].Hash(), items[1].Hash())
	assert.Equal(t, trailer.Hash("slug: duplicate"), items[0].Hash())
}

func TestScenarioDistinctTextHasDifferentHash(t *testing.T) {
	items := NewItems([]any{"Add a CSV serializer", "Add a JSON serializer"})
	assert.NotEqual(t, items[0].Hash(), items[1].Hash())
}
