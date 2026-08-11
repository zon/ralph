package project

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCommitLog struct {
	messages []string
	err      error
	base     string
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
	log := &stubCommitLog{messages: messages}
	out := &captureOutput{}
	return NewClient(log, out), log, out
}

func completedProject(values ...any) *Project {
	return &Project{Items: NewItems(values)}
}

func TestCompleteReportsIndexOnlyTrailer(t *testing.T) {
	c, _, _ := testClient("feat: add serializer\n\nRalph item 0 completed")
	indices, err := c.Complete(completedProject("a"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{0}, indices)
}

func TestCompleteReportsKeyedTrailerByIndex(t *testing.T) {
	c, _, _ := testClient("feat: export\n\nRalph item 3 (csv-serializer) completed")
	indices, err := c.Complete(completedProject("a", "b", "c", "d"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{3}, indices)
}

func TestCompleteCollectsTrailersAcrossCommits(t *testing.T) {
	c, _, _ := testClient(
		"Ralph item 1 completed",
		"Ralph item 2 (export-endpoint) completed",
	)
	indices, err := c.Complete(completedProject("a", "b", "c"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, indices)
}

func TestCompleteReadsAgainstSuppliedBase(t *testing.T) {
	c, log, _ := testClient("Ralph item 0 completed")
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

func TestCompleteAscendingAndDeduplicated(t *testing.T) {
	c, _, _ := testClient(
		"Ralph item 2 completed\nRalph item 0 completed",
		"Ralph item 3 (x) completed\nRalph item 2 completed",
	)
	indices, err := c.Complete(completedProject("a", "b", "c", "d", "e"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{0, 2, 3}, indices)
}

func TestCompleteKeyNeverUsedForMatching(t *testing.T) {
	c, _, _ := testClient("Ralph item 1 (some-other-key) completed")
	indices, err := c.Complete(completedProject("a", "b"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{1}, indices, "the index alone identifies the item")
}

func TestScenarioKeyMismatchHonoredByIndexWithWarning(t *testing.T) {
	c, _, out := testClient("Ralph item 2 (export-endpoint) completed")
	proj := completedProject(
		"a",
		"b",
		map[string]any{"slug": "csv-serializer"},
	)

	indices, err := c.Complete(proj, "main")
	require.NoError(t, err)
	assert.Equal(t, []int{2}, indices)

	warning := out.String()
	assert.Contains(t, warning, "export-endpoint")
	assert.Contains(t, warning, "csv-serializer")
}

func TestScenarioOutOfRangeIndexIgnoredWithWarning(t *testing.T) {
	c, _, out := testClient("Ralph item 5 completed")
	proj := completedProject("a", "b", "c")

	indices, err := c.Complete(proj, "main")
	require.NoError(t, err)
	assert.Empty(t, indices)
	assert.Contains(t, out.String(), "5")
	assert.Contains(t, out.String(), "outside")
}

func TestScenarioDuplicateTrailersCollapse(t *testing.T) {
	c, _, _ := testClient(
		"feat: a\n\nRalph item 1 completed",
		"feat: b\n\nRalph item 1 (x) completed",
	)
	indices, err := c.Complete(completedProject("a", "b", "c"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{1}, indices)
}

func TestCompleteNoItemArraySkipsRangeCheck(t *testing.T) {
	c, _, out := testClient("Ralph item 9 completed")
	indices, err := c.Complete(nil, "main")
	require.NoError(t, err)
	assert.Equal(t, []int{9}, indices)
	assert.Empty(t, out.String(), "no warning is emitted without a resolved array")
}

func TestCompleteNilItemsProjectSkipsRangeCheck(t *testing.T) {
	c, _, out := testClient("Ralph item 7 completed")
	indices, err := c.Complete(&Project{}, "main")
	require.NoError(t, err)
	assert.Equal(t, []int{7}, indices)
	assert.Empty(t, out.String())
}

func TestCompleteIndexOnlyTrailerOnKeyedItemNoWarning(t *testing.T) {
	c, _, out := testClient("Ralph item 0 completed")
	proj := completedProject(map[string]any{"slug": "csv-serializer"})

	indices, err := c.Complete(proj, "main")
	require.NoError(t, err)
	assert.Equal(t, []int{0}, indices)
	assert.Empty(t, out.String(), "an index-only trailer carries no key to mismatch")
}
