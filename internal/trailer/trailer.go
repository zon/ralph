// Package trailer formats and parses ralph completion trailers in commit
// messages. Trailer lines are pure strings with no filesystem, git, or network
// access, so they are unit tested directly.
package trailer

import (
	"fmt"
	"regexp"
	"strconv"
)

// Format renders the completion trailer line for a branch and an item index:
// "<branch>-<index>", for example "csv-export-2".
func Format(branch string, index int) string {
	return fmt.Sprintf("%s-%d", branch, index)
}

// trailerRe matches a whole line of the form "<branch>-<index>": a non-empty
// branch name and a trailing index joined by a hyphen. The branch is any run
// of git-branch characters, so a branch containing hyphens, dots, or slashes
// still parses; the trailing numeric segment is the index.
var trailerRe = regexp.MustCompile(`(?m)^([A-Za-z0-9][A-Za-z0-9._/-]*)-(\d+)\s*$`)

// Ref is one completion trailer extracted from a commit message: the branch
// and item index it names.
type Ref struct {
	Branch string
	Index  int
}

// Parse extracts every completion trailer from a commit message, returning the
// branch and index each one names, in order of appearance.
func Parse(message string) []Ref {
	matches := trailerRe.FindAllStringSubmatch(message, -1)
	refs := make([]Ref, 0, len(matches))
	for _, m := range matches {
		index, _ := strconv.Atoi(m[2])
		refs = append(refs, Ref{Branch: m[1], Index: index})
	}
	return refs
}
