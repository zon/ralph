package project

import (
	"errors"

	"github.com/zon/ralph/internal/projectfile"
)

// ErrExtraIterationsReached is returned when the iteration limit is exhausted but items are still incomplete.
var ErrExtraIterationsReached = errors.New("iteration limit reached")

// Project carries a resolved project file: the file path, the metadata derived
// from its document, the parsed document, and the resolved item array. It is
// never written back to YAML. The project file module preserves the parsed
// document instead. The yaml tags retain a readable rendering for prompts that
// fall back to marshaling when the raw document was not retained.
type Project struct {
	Slug       string                `yaml:"slug"`
	Title      string                `yaml:"title,omitempty"`
	Feature    string                `yaml:"feature,omitempty"`
	Items      []Item                `yaml:"-"`
	Path       string                `yaml:"-"`
	BaseBranch string                `yaml:"-"`
	Doc        *projectfile.Document `yaml:"-"`
}
