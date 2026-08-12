package projectfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

// Document is a parsed project file. It preserves the file's raw content so
// content outside the item array can be passed through untouched, and exposes
// the decoded document for item-query evaluation.
type Document struct {
	Raw  string
	Root any
}

// Parse reads a project file from disk and parses it as YAML or JSON, chosen by
// its extension. It returns an error for any other extension.
func Parse(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project file: %w", err)
	}

	doc := &Document{Raw: string(data)}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &doc.Root); err != nil {
			return nil, fmt.Errorf("failed to parse project YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &doc.Root); err != nil {
			return nil, fmt.Errorf("failed to parse project JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unrecognized input file type: %s", path)
	}
	return doc, nil
}

// ResolveItems evaluates a jq item query against a parsed document with gojq
// and returns the resolved item values.
//
// When the query produces exactly one output and that output is an array, the
// array's elements are the items. In every other case each output of the query
// is one item, so a query ending in `[]` and one returning the array itself
// resolve identically. Empty outputs are then discarded, so the result is
// either an empty slice or a slice whose every element is non-empty; see
// isEmptyItem for what counts as empty. Discarding happens before any caller
// indexes the result, so an item's index is its position among the survivors.
//
// Resolving nothing is not an error here: callers that need work to do report
// the empty slice themselves. Returns an error only when the query cannot be
// parsed or evaluated, naming the query.
func ResolveItems(doc *Document, query string) ([]any, error) {
	q, err := gojq.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("item query failed: %s: %w", query, err)
	}

	var items []any
	iter := q.Run(doc.Root)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("item query failed: %s: %w", query, err)
		}
		items = append(items, v)
	}

	if len(items) == 1 {
		if arr, ok := items[0].([]any); ok {
			items = arr
		}
	}

	resolved := make([]any, 0, len(items))
	for _, v := range items {
		if !isEmptyItem(v) {
			resolved = append(resolved, v)
		}
	}
	return resolved, nil
}

// isEmptyItem reports whether a query output carries no work and so is not an
// item. Empty means falsy: null, false, a zero number, a string that is empty
// or only whitespace, an empty mapping, or an empty sequence. Every other
// value is an item, whatever its shape.
func isEmptyItem(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool:
		return !rv.Bool()
	case reflect.String:
		return strings.TrimSpace(rv.String()) == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Map, reflect.Slice, reflect.Array:
		return rv.Len() == 0
	default:
		return false
	}
}

// Remove deletes a project file from disk.
func Remove(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove project file: %w", err)
	}
	return nil
}
