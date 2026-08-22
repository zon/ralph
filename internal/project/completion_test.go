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

func TestCompleteReportsBareTrailer(t *testing.T) {
	c, _, _ := testClient("feat: add serializer\n\ncsv-export-0")
	indices, err := c.Complete(completedProject("a"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{0}, indices)
}

func TestCompleteReportsTrailerByIndex(t *testing.T) {
	c, _, _ := testClient("feat: export\n\ncsv-export-3")
	indices, err := c.Complete(completedProject("a", "b", "c", "d"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{3}, indices)
}

func TestCompleteCollectsTrailersAcrossCommits(t *testing.T) {
	c, _, _ := testClient(
		"csv-export-1",
		"csv-export-2",
	)
	indices, err := c.Complete(completedProject("a", "b", "c"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, indices)
}

func TestCompleteReadsAgainstSuppliedBase(t *testing.T) {
	c, log, _ := testClient("csv-export-0")
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
		"csv-export-2\ncsv-export-0",
		"csv-export-3\ncsv-export-2",
	)
	indices, err := c.Complete(completedProject("a", "b", "c", "d", "e"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{0, 2, 3}, indices)
}

func TestScenarioOutOfRangeIndexIgnoredWithWarning(t *testing.T) {
	c, _, out := testClient("csv-export-5")
	proj := completedProject("a", "b", "c")

	indices, err := c.Complete(proj, "main")
	require.NoError(t, err)
	assert.Empty(t, indices)
	assert.Contains(t, out.String(), "5")
	assert.Contains(t, out.String(), "outside")
}

func TestScenarioDuplicateTrailersCollapse(t *testing.T) {
	c, _, _ := testClient(
		"feat: a\n\ncsv-export-1",
		"feat: b\n\ncsv-export-1",
	)
	indices, err := c.Complete(completedProject("a", "b", "c"), "main")
	require.NoError(t, err)
	assert.Equal(t, []int{1}, indices)
}

func TestCompleteNoItemArraySkipsRangeCheck(t *testing.T) {
	c, _, out := testClient("csv-export-9")
	indices, err := c.Complete(nil, "main")
	require.NoError(t, err)
	assert.Equal(t, []int{9}, indices)
	assert.Empty(t, out.String(), "no warning is emitted without a resolved array")
}

func TestCompleteNilItemsProjectSkipsRangeCheck(t *testing.T) {
	c, _, out := testClient("csv-export-7")
	indices, err := c.Complete(&Project{}, "main")
	require.NoError(t, err)
	assert.Equal(t, []int{7}, indices)
	assert.Empty(t, out.String())
}
