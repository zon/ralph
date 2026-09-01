package project

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/trailer"
)

type stubCommitLog struct {
	messages  []string
	err       error
	branch    string
	branchErr error
	base      string
}

func (s *stubCommitLog) CurrentBranch() (string, error) {
	return s.branch, s.branchErr
}

func (s *stubCommitLog) CommitMessages(base string) ([]string, error) {
	s.base = base
	return s.messages, s.err
}

type captureOutput struct {
	strings.Builder
}

func (c *captureOutput) Warnf(format string, a ...any) {
	fmt.Fprintf(c, format+"\n", a...)
}

func testClient(messages ...string) (*Client, *stubCommitLog, *captureOutput) {
	log := &stubCommitLog{messages: messages, branch: "csv-export"}
	out := &captureOutput{}
	return NewClient(log, out), log, out
}

func completedProject(values ...any) *Project {
	return &Project{Items: NewItems(values)}
}

// itemHash returns the completion hash of a plain-string item, matching what
// Item.Hash produces for an item whose value is that string.
func itemHash(v string) string {
	return trailer.Hash(v)
}

func TestCompleteReportsBareTrailer(t *testing.T) {
	c, _, _ := testClient("feat: add serializer\n\ncsv-export-" + itemHash("a"))
	hashes, err := c.Complete(completedProject("a"), "main")
	require.NoError(t, err)
	assert.Equal(t, []string{itemHash("a")}, hashes)
}

func TestCompleteReportsTrailerByHash(t *testing.T) {
	c, _, _ := testClient("feat: export\n\ncsv-export-" + itemHash("d"))
	hashes, err := c.Complete(completedProject("a", "b", "c", "d"), "main")
	require.NoError(t, err)
	assert.Equal(t, []string{itemHash("d")}, hashes)
}

func TestCompleteCollectsTrailersAcrossCommits(t *testing.T) {
	c, _, _ := testClient(
		"csv-export-"+itemHash("b"),
		"csv-export-"+itemHash("c"),
	)
	hashes, err := c.Complete(completedProject("a", "b", "c"), "main")
	require.NoError(t, err)
	assert.Equal(t, []string{itemHash("c"), itemHash("b")}, hashes, "trailers are sorted lexicographically")
}

func TestCompleteReadsAgainstSuppliedBase(t *testing.T) {
	c, log, _ := testClient("csv-export-" + itemHash("a"))
	_, err := c.Complete(completedProject("a"), "develop")
	require.NoError(t, err)
	assert.Equal(t, "develop", log.base)
}

func TestCompleteSurfacesCommitLogError(t *testing.T) {
	log := &stubCommitLog{err: fmt.Errorf("boom")}
	client := NewClient(log, &captureOutput{})
	_, err := client.Complete(completedProject("a"), "main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestCompleteSortedAndDeduplicated(t *testing.T) {
	c, _, _ := testClient(
		"csv-export-"+itemHash("c")+"\ncsv-export-"+itemHash("a"),
		"csv-export-"+itemHash("d")+"\ncsv-export-"+itemHash("c"),
	)
	hashes, err := c.Complete(completedProject("a", "b", "c", "d", "e"), "main")
	require.NoError(t, err)
	assert.Equal(t, []string{itemHash("d"), itemHash("c"), itemHash("a")}, hashes, "hashes are deduplicated and sorted lexicographically")
}

func TestScenarioUnmatchedHashIgnoredWithWarning(t *testing.T) {
	c, _, out := testClient("csv-export-" + itemHash("z"))
	proj := completedProject("a", "b", "c")

	hashes, err := c.Complete(proj, "main")
	require.NoError(t, err)
	assert.Empty(t, hashes)
	assert.Contains(t, out.String(), itemHash("z"))
	assert.Contains(t, out.String(), "matches no resolved item")
}

func TestScenarioDuplicateTrailersCollapse(t *testing.T) {
	c, _, _ := testClient(
		"feat: a\n\ncsv-export-"+itemHash("b"),
		"feat: b\n\ncsv-export-"+itemHash("b"),
	)
	hashes, err := c.Complete(completedProject("a", "b", "c"), "main")
	require.NoError(t, err)
	assert.Equal(t, []string{itemHash("b")}, hashes)
}

func TestCompleteNoItemArraySkipsResolvedItemCheck(t *testing.T) {
	c, _, out := testClient("csv-export-" + itemHash("z"))
	hashes, err := c.Complete(nil, "main")
	require.NoError(t, err)
	assert.Equal(t, []string{itemHash("z")}, hashes)
	assert.Empty(t, out.String(), "no warning is emitted without a resolved array")
}

func TestCompleteNilItemsProjectSkipsResolvedItemCheck(t *testing.T) {
	c, _, out := testClient("csv-export-" + itemHash("z"))
	hashes, err := c.Complete(&Project{}, "main")
	require.NoError(t, err)
	assert.Equal(t, []string{itemHash("z")}, hashes)
	assert.Empty(t, out.String())
}

func TestCompleteOnlyCountsCurrentBranchTrailers(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
		branch   string
		want     []string
	}{
		{
			name:     "mixed branches keep only the current branch",
			messages: []string{"csv-export-" + itemHash("a") + "\nother-branch-" + itemHash("b")},
			branch:   "csv-export",
			want:     []string{itemHash("a")},
		},
		{
			name:     "other branch trailers alone yield nothing",
			messages: []string{"other-branch-" + itemHash("a"), "another-branch-" + itemHash("c")},
			branch:   "csv-export",
			want:     []string{},
		},
		{
			name:     "current branch trailers still count",
			messages: []string{"csv-export-" + itemHash("b") + "\nother-branch-" + itemHash("d")},
			branch:   "csv-export",
			want:     []string{itemHash("b")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &stubCommitLog{messages: tt.messages, branch: tt.branch}
			c := NewClient(log, &captureOutput{})
			hashes, err := c.Complete(completedProject("a", "b", "c", "d", "e"), "main")
			require.NoError(t, err)
			assert.Equal(t, tt.want, hashes)
		})
	}
}

func TestCompleteOtherBranchTrailerIgnoredWithoutWarning(t *testing.T) {
	c, _, out := testClient("csv-export-"+itemHash("a"), "other-branch-"+itemHash("e"))
	hashes, err := c.Complete(completedProject("a", "b", "c"), "main")
	require.NoError(t, err)
	assert.Equal(t, []string{itemHash("a")}, hashes)
	assert.Empty(t, out.String(), "a trailer from another branch is ignored without a warning")
}

func TestCompleteNoItemArrayStillFiltersByBranch(t *testing.T) {
	c, _, out := testClient("other-branch-" + itemHash("z"))
	hashes, err := c.Complete(nil, "main")
	require.NoError(t, err)
	assert.Empty(t, hashes)
	assert.Empty(t, out.String())
}

func TestCompleteSurfacesCurrentBranchError(t *testing.T) {
	log := &stubCommitLog{branchErr: fmt.Errorf("detached head")}
	client := NewClient(log, &captureOutput{})
	_, err := client.Complete(completedProject("a"), "main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detached head")
}

func TestCompleteHashRoundTripsThroughBranchFilter(t *testing.T) {
	c, _, out := testClient("csv-export-" + itemHash("z"))
	hashes, err := c.Complete(nil, "main")
	require.NoError(t, err)
	assert.Equal(t, []string{itemHash("z")}, hashes)
	assert.Empty(t, out.String(), "no warning is emitted without a resolved array")
}

func TestCompleteBranchEndingInDigitStillParsesAndMatches(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		messages []string
		want     []string
	}{
		{
			name:     "trailing digit stays part of the branch",
			branch:   "csv-export-2",
			messages: []string{"csv-export-2-" + itemHash("c")},
			want:     []string{itemHash("c")},
		},
		{
			name:     "trailer with digit-ending branch does not match a shorter branch",
			branch:   "csv-export",
			messages: []string{"csv-export-2-" + itemHash("c")},
			want:     []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &stubCommitLog{messages: tt.messages, branch: tt.branch}
			c := NewClient(log, &captureOutput{})
			hashes, err := c.Complete(completedProject("a", "b", "c", "d"), "main")
			require.NoError(t, err)
			assert.Equal(t, tt.want, hashes)
		})
	}
}

func TestIncompleteSurfacesCurrentBranchError(t *testing.T) {
	log := &stubCommitLog{branchErr: fmt.Errorf("detached head")}
	client := NewClient(log, &captureOutput{})
	_, err := client.Incomplete(completedProject("a"), "main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detached head")
}
