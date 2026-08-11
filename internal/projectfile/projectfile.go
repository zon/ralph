package projectfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
// is one item. Returns an error naming the query when it cannot be evaluated,
// and a distinct error when it yields no output.
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

	if len(items) == 0 {
		return nil, fmt.Errorf("item query yielded no items: %s", query)
	}
	if len(items) == 1 {
		if arr, ok := items[0].([]any); ok {
			return arr, nil
		}
	}
	return items, nil
}
