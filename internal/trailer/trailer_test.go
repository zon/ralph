package trailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatJoinsBranchAndIndex(t *testing.T) {
	assert.Equal(t, "csv-export-3", Format("csv-export", 3))
}

func TestFormatZeroIndex(t *testing.T) {
	assert.Equal(t, "csv-export-0", Format("csv-export", 0))
}

func TestFormatBranchContainingHyphen(t *testing.T) {
	assert.Equal(t, "feature/csv-export-2", Format("feature/csv-export", 2))
}

func TestParseBareTrailer(t *testing.T) {
	refs := Parse("csv-export-2")
	require.Len(t, refs, 1)
	assert.Equal(t, "csv-export", refs[0].Branch)
	assert.Equal(t, 2, refs[0].Index)
}

func TestParseTrailerInsideCommitMessage(t *testing.T) {
	refs := Parse("feat: add csv serializer\n\nImplementation details.\n\ncsv-export-3")
	require.Len(t, refs, 1)
	assert.Equal(t, "csv-export", refs[0].Branch)
	assert.Equal(t, 3, refs[0].Index)
}

func TestParseTrailerWithoutTrailingNewline(t *testing.T) {
	refs := Parse("feat: done\n\ncsv-export-2")
	require.Len(t, refs, 1)
	assert.Equal(t, 2, refs[0].Index)
}

func TestParseTrailerWithTrailingWhitespace(t *testing.T) {
	refs := Parse("csv-export-1   \n")
	require.Len(t, refs, 1)
	assert.Equal(t, 1, refs[0].Index)
}

func TestParseMultipleTrailers(t *testing.T) {
	refs := Parse("feat: export\n\ncsv-export-1\ncsv-export-2")
	require.Len(t, refs, 2)
	assert.Equal(t, "csv-export", refs[0].Branch)
	assert.Equal(t, 1, refs[0].Index)
	assert.Equal(t, "csv-export", refs[1].Branch)
	assert.Equal(t, 2, refs[1].Index)
}

func TestParseBranchWithSlash(t *testing.T) {
	refs := Parse("feature/csv-export-2")
	require.Len(t, refs, 1)
	assert.Equal(t, "feature/csv-export", refs[0].Branch)
	assert.Equal(t, 2, refs[0].Index)
}

func TestParseBranchContainingDigit(t *testing.T) {
	refs := Parse("csv-2-3")
	require.Len(t, refs, 1)
	assert.Equal(t, "csv-2", refs[0].Branch)
	assert.Equal(t, 3, refs[0].Index, "the trailing numeric segment is the index")
}

func TestParseMessageWithoutTrailer(t *testing.T) {
	assert.Empty(t, Parse("feat: fix bug\n\nRefactor the parser."))
}

func TestParseIgnoresTrailerMentionedInProse(t *testing.T) {
	refs := Parse("README: the line `csv-export-0` marks an item done")
	assert.Empty(t, refs)
}

func TestParseIgnoresMalformedTrailers(t *testing.T) {
	for _, message := range []string{
		"csv-export",
		"-2",
		"csv-export-two",
		"csv-export-2 extra",
		"csv-export-2 marks it done",
		"Ralph item 0 completed",
		" csv-export-2",
		"2",
	} {
		assert.Empty(t, Parse(message), "message %q should not parse", message)
	}
}
