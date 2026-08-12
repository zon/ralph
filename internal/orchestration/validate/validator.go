package validate

import (
	"bytes"
	"fmt"

	"github.com/zon/ralph/internal/projectfile"
)

const MaxAttempts = 10

var (
	ErrNoChange    = fmt.Errorf("agent made no changes to the project file")
	ErrUnreachable = fmt.Errorf("unreachable: validate loop exited without returning")
)

// ProjectFile reads and evaluates a project file. The real implementation
// composes the project file module's parse, item-query evaluation, and raw
// reads; validation performs no file work itself.
type ProjectFile interface {
	Parse(path string) (*projectfile.Document, error)
	ResolveItems(doc *projectfile.Document, query string) ([]any, error)
	ReadFile(path string) ([]byte, error)
}

type AgentClient interface {
	FixProject(path string, parseErr error, model string) error
}

// Result reports a successful validation: the validated file path and the
// number of items the query resolved.
type Result struct {
	Path      string
	ItemCount int
}

type Validator struct {
	file  ProjectFile
	agent AgentClient
	model string
}

// Validate performs exactly three checks, in order: the file parses as YAML or
// JSON, the item query evaluates against the parsed document without error, and
// the query resolves to at least one item. There is no schema check — no field
// is required and no field is rejected. A parse failure enters the bounded fix
// loop; a query that cannot be evaluated or yields no items is returned as an
// error without invoking the agent.
func (v *Validator) Validate(path, query string) (*Result, error) {
	if query == "" {
		query = "."
	}
	doc, err := v.parse(path)
	if err != nil {
		return nil, err
	}
	items, err := v.file.ResolveItems(doc, query)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("item query yielded no items: %s", query)
	}
	return &Result{Path: path, ItemCount: len(items)}, nil
}

// parse reads and parses the file at path, entering the bounded fix loop only
// when the file fails to parse. The agent repairs the file in place and parsing
// is retried against the updated content until it succeeds or the attempt limit
// is reached.
func (v *Validator) parse(path string) (*projectfile.Document, error) {
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		doc, parseErr := v.file.Parse(path)
		if parseErr == nil {
			return doc, nil
		}
		if attempt == MaxAttempts {
			return nil, parseErr
		}
		before, _ := v.file.ReadFile(path)
		if err := v.agent.FixProject(path, parseErr, v.model); err != nil {
			return nil, err
		}
		after, _ := v.file.ReadFile(path)
		if bytes.Equal(before, after) {
			return nil, ErrNoChange
		}
	}
	return nil, ErrUnreachable
}
