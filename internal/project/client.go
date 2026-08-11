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

func (c *Client) Load(path string) (*Project, error) {
	return LoadProject(path)
}

func (c *Client) ResolveInputFile(path string) (*InputFile, error) {
	return ResolveInputFile(path)
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

func (c *Client) Reload(proj *Project) *Project {
	if proj.Path != "" {
		if latest, err := LoadProject(proj.Path); err == nil {
			return latest
		}
	}
	return proj
}

func (c *Client) AllRequirementsPassing(proj *Project) bool {
	allComplete, _, _ := CheckCompletion(proj)
	return allComplete
}

func (c *Client) ExtraIterations(proj *Project, cfg *config.RalphConfig) int {
	if cfg.ExtraIterations != nil {
		return *cfg.ExtraIterations
	}
	count := len(proj.Requirements)
	extra := int(math.Ceil(float64(count) * 0.2))
	return extra
}

func (c *Client) ExtraIterationsError(proj *Project) error {
	_, _, failingCount := CheckCompletion(proj)
	return fmt.Errorf("%w: %d requirements still failing", ErrExtraIterationsReached, failingCount)
}

func (c *Client) HasChanges(proj *Project) bool {
	return git.HasFileChanges(proj.Path)
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

func (c *Client) NormalizeAndStage(proj *Project) {
	data, err := os.ReadFile(proj.Path)
	if err != nil {
		return
	}
	normalized := []byte(strings.TrimRight(string(data), "\n") + "\n")
	if len(normalized) != len(data) {
		os.WriteFile(proj.Path, normalized, 0644)
	}
	git.StageFile(proj.Path)
}
