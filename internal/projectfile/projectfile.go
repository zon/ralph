package projectfile

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

// Document is a parsed project file. It preserves the file's raw content so
// content outside the item array can be passed through untouched, retains the
// parsed YAML node so the file can be rewritten canonically with its key order
// intact, and exposes the decoded document for item-query evaluation.
type Document struct {
	Raw  string
	Root any
	Node *yaml.Node
}

// Parse reads a project file from disk and parses it as YAML or JSON, chosen by
// its extension. JSON is parsed with the YAML parser because every JSON value
// is also a YAML value, so both formats produce the same document and a JSON
// input can be rewritten as YAML without losing key order. It returns an error
// for any other extension.
func Parse(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project file: %w", err)
	}

	doc := &Document{Raw: string(data)}
	var node yaml.Node
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &node); err != nil {
			return nil, fmt.Errorf("failed to parse project YAML: %w", err)
		}
	case ".json":
		if err := yaml.Unmarshal(data, &node); err != nil {
			return nil, fmt.Errorf("failed to parse project JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unrecognized input file type: %s", path)
	}
	if err := node.Decode(&doc.Root); err != nil {
		return nil, fmt.Errorf("failed to parse project file: %w", err)
	}
	doc.Node = &node
	return doc, nil
}

// CanonicalPath returns the path WriteCanonical writes to for the given input
// path: a .json input is rewritten as a sibling .yaml file, any other input is
// rewritten in place.
func CanonicalPath(path string) string {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		return strings.TrimSuffix(path, filepath.Ext(path)) + ".yaml"
	}
	return path
}

// WriteCanonical writes a parsed document back to disk as canonical YAML,
// preserving its content and key order. Writing a document parsed from a .json
// path produces a sibling .yaml file; removing the original is the caller's
// decision.
func WriteCanonical(path string, doc *Document) error {
	target := CanonicalPath(path)
	node := doc.Node
	if node == nil {
		data, err := yaml.Marshal(doc.Root)
		if err != nil {
			return fmt.Errorf("failed to marshal canonical YAML: %w", err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("failed to write canonical YAML: %w", err)
		}
		return nil
	}
	normalizeNode(node)
	data, err := yaml.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal canonical YAML: %w", err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("failed to write canonical YAML: %w", err)
	}
	return nil
}

// normalizeNode rewrites a parsed node's styles in place so marshaling produces
// canonical block YAML: mappings and sequences are block-laid, and scalars are
// left for the marshaler to quote only when required. Flow styles introduced by
// JSON parsing are dropped, but already-canonical block input is untouched, so
// a canonical file round-trips byte-identically.
func normalizeNode(n *yaml.Node) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.MappingNode, yaml.SequenceNode:
		n.Style = 0
	case yaml.ScalarNode:
		if n.Style == yaml.DoubleQuotedStyle || n.Style == yaml.SingleQuotedStyle {
			n.Style = 0
		}
	}
	for _, c := range n.Content {
		normalizeNode(c)
	}
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
