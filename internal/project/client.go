package project

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/projectfile"
	"github.com/zon/ralph/internal/trailer"
)

// CommitLog retrieves the commit messages of the commits on the current branch
// that are not on the base branch.
type CommitLog interface {
	CommitMessages(base string) ([]string, error)
}

// WarnOutput emits warnings during completion reconciliation.
type WarnOutput interface {
	Warnf(format string, a ...any)
}

type Client struct {
	log CommitLog
	out WarnOutput
}

func NewClient(log CommitLog, out WarnOutput) *Client {
	return &Client{log: log, out: out}
}

// Resolve parses the project file at path and evaluates the item query against
// it, returning a Project carrying the file path, slug, title, parsed document,
// and resolved item array. It reads and parses nothing itself, delegating file
// reading, parsing, and query evaluation, then builds the domain values from
// the results. Returns an error when the file does not parse, the query fails,
// or the query yields no items — resolution discards empty outputs, so a query
// that produces nothing but empty values leaves no work to run.
func (c *Client) Resolve(path string, query string) (*Project, error) {
	doc, err := projectfile.Parse(path)
	if err != nil {
		return nil, err
	}
	values, err := projectfile.ResolveItems(doc, query)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("item query yielded no items: %s", query)
	}
	slug := slugFrom(doc, path)
	return &Project{
		Slug:  slug,
		Title: titleFrom(doc, slug),
		Path:  path,
		Items: NewItems(values),
		Doc:   doc,
	}, nil
}

// slugFrom derives the project slug from the document's top-level `slug` field
// when the top level is a mapping carrying one, and otherwise from the input
// file's base name without its extension.
func slugFrom(doc *projectfile.Document, path string) string {
	if m, ok := doc.Root.(map[string]any); ok {
		if v, ok := m["slug"].(string); ok && v != "" {
			return v
		}
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// titleFrom derives the project title from the document's top-level `title`
// field when present, and otherwise from the resolved slug.
func titleFrom(doc *projectfile.Document, slug string) string {
	if m, ok := doc.Root.(map[string]any); ok {
		if v, ok := m["title"].(string); ok && v != "" {
			return v
		}
	}
	return slug
}

// Complete reads the completion trailers from the commit messages on the
// current branch that are not on the base branch and returns the ascending,
// deduplicated indices they name. When proj is non-nil and its item array was
// resolved, an index outside the array is dropped with a warning naming it, and
// a trailer whose key differs from the key of the item at its index warns
// without dropping the index. When no item array was resolved, every trailer
// found in the log is reported without a range check.
func (c *Client) Complete(proj *Project, base string) ([]int, error) {
	messages, err := c.log.CommitMessages(base)
	if err != nil {
		return nil, err
	}
	var refs []trailer.Ref
	for _, m := range messages {
		refs = append(refs, trailer.Parse(m)...)
	}
	return c.reconcile(refs, proj), nil
}

// reconcile turns parsed completion trailers into the reported indices, applying
// the range and key checks only when an item array was resolved.
func (c *Client) reconcile(refs []trailer.Ref, proj *Project) []int {
	resolved := proj != nil && proj.Items != nil
	seen := make(map[int]struct{}, len(refs))
	for _, r := range refs {
		if resolved {
			if r.Index >= len(proj.Items) {
				c.out.Warnf("completion trailer names index %d which is outside the resolved item array (%d items); ignoring", r.Index, len(proj.Items))
				continue
			}
			if r.Key != "" && r.Key != proj.Items[r.Index].Key() {
				c.out.Warnf("completion trailer names key %q for item %d, but the item's key is %q; honoring index %d", r.Key, r.Index, proj.Items[r.Index].Key(), r.Index)
			}
		}
		seen[r.Index] = struct{}{}
	}
	indices := make([]int, 0, len(seen))
	for i := range seen {
		indices = append(indices, i)
	}
	sort.Ints(indices)
	return indices
}

// Incomplete returns the items of proj.Items whose indices are not reported
// complete in the branch commit log, in array order. An empty result is the
// iteration loop's exit condition.
func (c *Client) Incomplete(proj *Project, base string) ([]Item, error) {
	complete, err := c.Complete(proj, base)
	if err != nil {
		return nil, err
	}
	completeSet := make(map[int]struct{}, len(complete))
	for _, i := range complete {
		completeSet[i] = struct{}{}
	}
	incomplete := make([]Item, 0, len(proj.Items))
	for _, it := range proj.Items {
		if _, ok := completeSet[it.Index]; !ok {
			incomplete = append(incomplete, it)
		}
	}
	return incomplete, nil
}

// IncompleteError returns an error naming the items that are still incomplete.
func (c *Client) IncompleteError(proj *Project, base string) error {
	incomplete, err := c.Incomplete(proj, base)
	if err != nil {
		return err
	}
	if len(incomplete) == 0 {
		return nil
	}
	names := make([]string, 0, len(incomplete))
	for _, it := range incomplete {
		name := fmt.Sprintf("item %d", it.Index)
		if key := it.Key(); key != "" {
			name = fmt.Sprintf("item %d (%s)", it.Index, key)
		}
		names = append(names, name)
	}
	return fmt.Errorf("%w: %s still incomplete", ErrExtraIterationsReached, strings.Join(names, ", "))
}

// Remove deletes the project file and stages the deletion, delegating the
// filesystem work to the project file module and the git wrapper.
func (c *Client) Remove(proj *Project) error {
	if err := projectfile.Remove(proj.Path); err != nil {
		return err
	}
	return git.StageFile(proj.Path)
}

func (c *Client) ValidateFile(path string) error {
	if path == "" {
		return fmt.Errorf("project file required (see --help)")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve project file path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("project file not found: %s", absPath)
	}
	return nil
}

func (c *Client) ExtraIterations(proj *Project, cfg *config.RalphConfig) int {
	if cfg.ExtraIterations != nil {
		return *cfg.ExtraIterations
	}
	count := len(proj.Items)
	extra := int(math.Ceil(float64(count) * 0.2))
	return extra
}

func (c *Client) HasSpec(proj *Project) bool {
	return proj.Feature != ""
}

func (c *Client) HasOrchestration(proj *Project) bool {
	if proj.Feature == "" {
		return false
	}
	repoRoot, err := git.FindRepoRoot()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(repoRoot, proj.Feature, "orchestration.md"))
	return err == nil
}

func (c *Client) RemoveOrchestration(proj *Project) error {
	repoRoot, err := git.FindRepoRoot()
	if err != nil {
		return err
	}
	orchestrationPath := filepath.Join(repoRoot, proj.Feature, "orchestration.md")
	if err := os.Remove(orchestrationPath); err != nil {
		return err
	}
	return git.StageFile(orchestrationPath)
}
