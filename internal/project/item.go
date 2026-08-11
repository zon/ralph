package project

import "fmt"

// Item is one element of a project's resolved item array.
//
// Index is the item's 0-based position in the array and the only identifier
// ralph uses. Value is the raw value the item was resolved from, unchanged, so
// it can be handed to an agent verbatim. Key is an optional convenience label
// derived from Value; it is never used to identify or match items.
type Item struct {
	Index int
	Value any
}

// NewItems wraps resolved values into items, assigning each value its 0-based
// index in the array and deriving its key. Keys are never checked for
// uniqueness: two items may share a key without error.
func NewItems(values []any) []Item {
	items := make([]Item, len(values))
	for i, v := range values {
		items[i] = Item{Index: i, Value: v}
	}
	return items
}

// Key returns the item's key: the scalar `slug`, `id`, or `name` field of a
// mapping, checked in that order, rendered as a string. An item that is not a
// mapping, or a mapping with none of those fields, has no key and Key returns
// an empty string.
func (it Item) Key() string {
	m, ok := it.Value.(map[string]any)
	if !ok {
		return ""
	}
	for _, field := range []string{"slug", "id", "name"} {
		if v, ok := m[field]; ok && isScalar(v) {
			return fmt.Sprint(v)
		}
	}
	return ""
}

// isScalar reports whether v is a scalar value rather than a collection or
// null, so it can serve as an item key.
func isScalar(v any) bool {
	switch v.(type) {
	case nil, map[string]any, map[any]any, []any:
		return false
	}
	return true
}
