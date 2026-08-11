// Package trailer formats and parses ralph completion trailers in commit
// messages. Trailer lines are pure strings with no filesystem, git, or network
// access, so they are unit tested directly.
package trailer

import (
	"fmt"
	"regexp"
	"strconv"
)

// Format renders the completion trailer line for an item index and its
// optional key: "Ralph item 2 completed" or "Ralph item 2 (export-endpoint)
// completed".
func Format(index int, key string) string {
	if key == "" {
		return fmt.Sprintf("Ralph item %d completed", index)
	}
	return fmt.Sprintf("Ralph item %d (%s) completed", index, key)
}

// trailerRe matches a whole line of the form "Ralph item <index> completed" or
// "Ralph item <index> (<key>) completed". A key is a non-empty run of any
// character except a closing parenthesis.
var trailerRe = regexp.MustCompile(`(?m)^Ralph item (\d+)(?: \(([^)]+)\))? completed\s*$`)

// Ref is one completion trailer extracted from a commit message: the item
// index it names and any key carried alongside it.
type Ref struct {
	Index int
	Key   string
}

// Parse extracts every completion trailer from a commit message, returning the
// index and any key each one names, in order of appearance.
func Parse(message string) []Ref {
	matches := trailerRe.FindAllStringSubmatch(message, -1)
	refs := make([]Ref, 0, len(matches))
	for _, m := range matches {
		index, _ := strconv.Atoi(m[1])
		refs = append(refs, Ref{Index: index, Key: m[2]})
	}
	return refs
}
