package project

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/zon/ralph/internal/config"
)

// WithItems returns a project whose resolved item array holds exactly n items.
// Each item carries its 0-based array position as its index and its plain
// string value, so the project is a valid fixture for iteration and
// completion tests.
func WithItems(n int) *Project {
	values := make([]any, n)
	for i := range values {
		values[i] = fmt.Sprintf("item-%d", i)
	}
	return &Project{
		Slug:  "test-project",
		Title: "Test Project",
		Items: NewItems(values),
	}
}

// Any returns a valid project in a default state: a resolved item array of
// three items with a stable slug and title.
func Any() *Project {
	return WithItems(3)
}

// MockProject is a project client factory for tests. It describes completion
// through item arrays: Resolve returns the project, and Complete and
// Incomplete answer from the configured completion behavior. It records its
// call history so tests can assert the item query used, the base branch used,
// and how often resolution and completion queries were made.
type MockProject struct {
	proj                   *Project
	resolveErr             error
	allComplete            bool
	alwaysIncomplete       bool
	incompleteUntil        int
	completeIndices        []int
	thenAllComplete        bool
	hasSpec                bool
	hasOrchestration       bool
	removeOrchestrationErr error
	removeErr              error

	lastQuery            string
	lastPath             string
	lastBase             string
	resolveCount         int
	incompleteCount      int
	written              bool
	removed              bool
	orchestrationRemoved bool
}

// ThatReportsAllComplete returns a project client whose Incomplete always
// returns an empty slice, so a run records every item complete.
func ThatReportsAllComplete() *MockProject {
	return &MockProject{allComplete: true}
}

// ThatReportsIncompleteUntil returns a project client whose Incomplete
// returns a non-empty slice for the first n calls and an empty slice
// thereafter, so a run completes after n iterations.
func ThatReportsIncompleteUntil(n int) *MockProject {
	return &MockProject{incompleteUntil: n}
}

// ThatAlwaysReportsIncomplete returns a project client whose Incomplete
// always returns a non-empty slice, so a run only ends when the iteration
// limit is reached.
func ThatAlwaysReportsIncomplete() *MockProject {
	return &MockProject{alwaysIncomplete: true}
}

// ThatReportsComplete returns a project client whose Complete returns the
// given indices, so Incomplete returns the rest of the resolved array.
func ThatReportsComplete(indices ...int) *MockProject {
	return &MockProject{completeIndices: indices}
}

// ThatFailsResolution returns a project client whose Resolve fails with an
// item-query error.
func ThatFailsResolution() *MockProject {
	return &MockProject{resolveErr: errors.New("item query yielded no items: .")}
}

// ThenAllComplete chains a modifier so the second and later Incomplete calls
// report every item complete, ending the loop after one iteration.
func (m *MockProject) ThenAllComplete() *MockProject {
	m.thenAllComplete = true
	return m
}

// WithNoSpec chains a modifier so HasSpec returns false.
func (m *MockProject) WithNoSpec() *MockProject {
	m.hasSpec = false
	return m
}

// WithSpecButNoOrchestration chains a modifier so HasSpec returns true and
// HasOrchestration returns false.
func (m *MockProject) WithSpecButNoOrchestration() *MockProject {
	m.hasSpec = true
	m.hasOrchestration = false
	return m
}

// WithOrchestration chains a modifier so HasSpec and HasOrchestration both
// return true.
func (m *MockProject) WithOrchestration() *MockProject {
	m.hasSpec = true
	m.hasOrchestration = true
	return m
}

// ThatFailsRemoval chains a modifier so RemoveOrchestration returns an error.
func (m *MockProject) ThatFailsRemoval() *MockProject {
	m.removeOrchestrationErr = errors.New("orchestration removal failed")
	return m
}

// ThatFailsProjectRemoval chains a modifier so Remove returns an error.
func (m *MockProject) ThatFailsProjectRemoval() *MockProject {
	m.removeErr = errors.New("project removal failed")
	return m
}

// WithResolvedItems sets the item count of the project Resolve returns, so
// iteration limits derived from the resolved array match the fixture.
func (m *MockProject) WithResolvedItems(n int) *MockProject {
	m.proj = WithItems(n)
	return m
}

// Resolve records the item query and the file path, and returns the client's
// project, or the configured item-query error.
func (m *MockProject) Resolve(path string, query string) (*Project, error) {
	m.resolveCount++
	m.lastQuery = query
	m.lastPath = path
	if m.resolveErr != nil {
		return nil, m.resolveErr
	}
	if m.proj == nil {
		m.proj = Any()
	}
	return m.proj, nil
}

// Complete records the base branch and returns the hashes of the configured
// complete indices' items.
func (m *MockProject) Complete(proj *Project, base string) ([]string, error) {
	m.lastBase = base
	hashes := make([]string, 0, len(m.completeIndices))
	for _, i := range m.completeIndices {
		hashes = append(hashes, proj.Items[i].Hash())
	}
	return hashes, nil
}

// Incomplete records the base branch and returns the items the configured
// completion behavior leaves incomplete, in array order.
func (m *MockProject) Incomplete(proj *Project, base string) ([]Item, error) {
	m.incompleteCount++
	m.lastBase = base
	switch {
	case m.allComplete:
		return nil, nil
	case m.alwaysIncomplete:
		return cloneItems(proj.Items), nil
	case m.incompleteUntil > 0:
		if m.incompleteCount > m.incompleteUntil {
			return nil, nil
		}
		return cloneItems(proj.Items), nil
	case m.completeIndices != nil:
		if m.thenAllComplete && m.incompleteCount > 1 {
			return nil, nil
		}
		complete := make(map[string]struct{}, len(m.completeIndices))
		for _, i := range m.completeIndices {
			complete[proj.Items[i].Hash()] = struct{}{}
		}
		incomplete := make([]Item, 0, len(proj.Items))
		for _, it := range proj.Items {
			if _, ok := complete[it.Hash()]; !ok {
				incomplete = append(incomplete, it)
			}
		}
		return incomplete, nil
	}
	return nil, nil
}

// ExtraIterations returns the configured extra iteration count, or 20% of the
// item count rounded up when unset.
func (m *MockProject) ExtraIterations(proj *Project, cfg *config.RalphConfig) int {
	if cfg.ExtraIterations != nil {
		return *cfg.ExtraIterations
	}
	return int(math.Ceil(float64(len(proj.Items)) * 0.2))
}

// IncompleteError returns an error naming the items that are still incomplete,
// delegating the incomplete computation to Incomplete.
func (m *MockProject) IncompleteError(proj *Project, base string) error {
	incomplete, err := m.Incomplete(proj, base)
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

// Remove records that the project file was removed and returns the configured
// removal error.
func (m *MockProject) Remove(proj *Project) error {
	m.removed = true
	return m.removeErr
}

// Write records that a write to the project file was attempted.
func (m *MockProject) Write(proj *Project) {
	m.written = true
}

func (m *MockProject) HasSpec(proj *Project) bool {
	return m.hasSpec
}

func (m *MockProject) HasOrchestration(proj *Project) bool {
	return m.hasOrchestration
}

// RemoveOrchestration records that the orchestration document was removed and
// returns the configured removal error.
func (m *MockProject) RemoveOrchestration(proj *Project) error {
	m.orchestrationRemoved = true
	return m.removeOrchestrationErr
}

// LastQuery returns the item query passed to the most recent Resolve call.
func (m *MockProject) LastQuery() string {
	return m.lastQuery
}

// LastPath returns the file path passed to the most recent Resolve call.
func (m *MockProject) LastPath() string {
	return m.lastPath
}

// LastBase returns the base branch passed to the most recent completion call.
func (m *MockProject) LastBase() string {
	return m.lastBase
}

// ResolveCount returns the number of times Resolve was called.
func (m *MockProject) ResolveCount() int {
	return m.resolveCount
}

// IncompleteCallCount returns the number of times Incomplete was called.
func (m *MockProject) IncompleteCallCount() int {
	return m.incompleteCount
}

// Written returns whether a write to the project file was attempted.
func (m *MockProject) Written() bool {
	return m.written
}

// Removed returns whether Remove was called.
func (m *MockProject) Removed() bool {
	return m.removed
}

// OrchestrationRemoved returns whether RemoveOrchestration was called.
func (m *MockProject) OrchestrationRemoved() bool {
	return m.orchestrationRemoved
}

func cloneItems(items []Item) []Item {
	cloned := make([]Item, len(items))
	copy(cloned, items)
	return cloned
}
