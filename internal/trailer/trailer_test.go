package trailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashReturnsSevenBase62Chars(t *testing.T) {
	for _, text := range []string{
		"a",
		"Add a CSV serializer",
		"description: Build the export endpoint",
		"",
		"   ",
	} {
		h := Hash(text)
		require.Len(t, h, 7, "hash of %q must be exactly 7 characters", text)
		for _, c := range h {
			assert.Contains(t, base62Alphabet, string(c), "hash of %q contains a non-base-62 character", text)
		}
	}
}

func TestHashIsDeterministic(t *testing.T) {
	text := "Add a CSV serializer"
	assert.Equal(t, Hash(text), Hash(text))
}

func TestHashIgnoresCaseAndSurroundingWhitespace(t *testing.T) {
	assert.Equal(t, Hash("  Item 0  "), Hash("item 0"))
	assert.Equal(t, Hash("CSV Export"), Hash("csv export"))
}

func TestHashDiffersAcrossDistinctTexts(t *testing.T) {
	a := Hash("Add a CSV serializer")
	b := Hash("Add a JSON serializer")
	assert.NotEqual(t, a, b)
}

func TestFormatJoinsBranchAndHash(t *testing.T) {
	assert.Equal(t, "csv-export-IYAWN02", Format("csv-export", "IYAWN02"))
}

func TestFormatBranchContainingHyphen(t *testing.T) {
	assert.Equal(t, "feature/csv-export-IYAWN02", Format("feature/csv-export", "IYAWN02"))
}

func TestParseBareTrailer(t *testing.T) {
	refs := Parse("csv-export-IYAWN02")
	require.Len(t, refs, 1)
	assert.Equal(t, "csv-export", refs[0].Branch)
	assert.Equal(t, "IYAWN02", refs[0].Hash)
}

func TestParseTrailerInsideCommitMessage(t *testing.T) {
	refs := Parse("feat: add csv serializer\n\nImplementation details.\n\ncsv-export-7FhX6dT")
	require.Len(t, refs, 1)
	assert.Equal(t, "csv-export", refs[0].Branch)
	assert.Equal(t, "7FhX6dT", refs[0].Hash)
}

func TestParseTrailerWithoutTrailingNewline(t *testing.T) {
	refs := Parse("feat: done\n\ncsv-export-IYAWN02")
	require.Len(t, refs, 1)
	assert.Equal(t, "IYAWN02", refs[0].Hash)
}

func TestParseTrailerWithTrailingWhitespace(t *testing.T) {
	refs := Parse("csv-export-1Q5RvYo   \n")
	require.Len(t, refs, 1)
	assert.Equal(t, "1Q5RvYo", refs[0].Hash)
}

func TestParseMultipleTrailers(t *testing.T) {
	refs := Parse("feat: export\n\ncsv-export-IYAWN02\ncsv-export-7FhX6dT")
	require.Len(t, refs, 2)
	assert.Equal(t, "csv-export", refs[0].Branch)
	assert.Equal(t, "IYAWN02", refs[0].Hash)
	assert.Equal(t, "csv-export", refs[1].Branch)
	assert.Equal(t, "7FhX6dT", refs[1].Hash)
}

func TestParseBranchWithSlash(t *testing.T) {
	refs := Parse("feature/csv-export-IYAWN02")
	require.Len(t, refs, 1)
	assert.Equal(t, "feature/csv-export", refs[0].Branch)
	assert.Equal(t, "IYAWN02", refs[0].Hash)
}

func TestParseBranchContainingDigit(t *testing.T) {
	refs := Parse("csv-2-IYAWN02")
	require.Len(t, refs, 1)
	assert.Equal(t, "csv-2", refs[0].Branch)
	assert.Equal(t, "IYAWN02", refs[0].Hash)
}

func TestParseMessageWithoutTrailer(t *testing.T) {
	assert.Empty(t, Parse("feat: fix bug\n\nRefactor the parser."))
}

func TestParseIgnoresTrailerMentionedInProse(t *testing.T) {
	refs := Parse("README: the line `csv-export-IYAWN02` marks an item done")
	assert.Empty(t, refs)
}

func TestParseIgnoresMalformedTrailers(t *testing.T) {
	for _, message := range []string{
		"csv-export",
		"-IYAWN02",
		"csv-export-two",
		"csv-export-IYAWN02 extra",
		"csv-export-IYAWN02 marks it done",
		"Ralph item 0 completed",
		" csv-export-IYAWN02",
		"IYAWN02",
		"csv-export-0",
		"csv-export-23",
		"csv-export-ab12cd",
		"csv-export-abcdefgh",
	} {
		assert.Empty(t, Parse(message), "message %q should not parse", message)
	}
}

func TestHashMatchesFormatTrailer(t *testing.T) {
	trailer := Format("csv-export", Hash("item 0"))
	refs := Parse(trailer)
	require.Len(t, refs, 1)
	assert.Equal(t, "csv-export", refs[0].Branch)
	assert.Equal(t, Hash("item 0"), refs[0].Hash)
}
