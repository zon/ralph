package project

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zon/ralph/internal/trailer"
)

// Item is one element of a project's resolved item array.
//
// Index is the item's 0-based position in the array, used for ordering and
// display. Value is the raw value the item was resolved from, unchanged, so
// it can be handed to an agent verbatim. Key is an optional convenience label
// derived from Value; it is never used to identify or match items. Completion
// identifies items by Hash, a stable hash of the item's text, so an item is
// the same item across runs as long as its text is unchanged.
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

// Text renders the item's raw value as canonical YAML text, so the same value
// always produces the same text and therefore the same completion hash.
func (it Item) Text() string {
	data, err := yaml.Marshal(it.Value)
	if err != nil {
		return fmt.Sprint(it.Value)
	}
	return strings.TrimRight(string(data), "\n")
}

// Hash returns the item's completion hash: a 7-character base-62 hash of the
// item's canonical text. Two items whose text is identical share a hash.
func (it Item) Hash() string {
	return trailer.Hash(it.Text())
}

// ItemValues returns the raw value of each item, in array order.
func ItemValues(items []Item) []any {
	values := make([]any, len(items))
	for i, it := range items {
		values[i] = it.Value
	}
	return values
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
