package trailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatWithKey(t *testing.T) {
	assert.Equal(t, "Ralph item 3 (csv-serializer) completed", Format(3, "csv-serializer"))
}

func TestFormatWithoutKey(t *testing.T) {
	assert.Equal(t, "Ralph item 0 completed", Format(0, ""))
}

func TestFormatRendersIndex(t *testing.T) {
	assert.Equal(t, "Ralph item 2 completed", Format(2, ""))
}

func TestParseIndexOnlyTrailer(t *testing.T) {
	refs := Parse("Ralph item 0 completed")
	require.Len(t, refs, 1)
	assert.Equal(t, 0, refs[0].Index)
	assert.Empty(t, refs[0].Key)
}

func TestParseTrailerWithKey(t *testing.T) {
	refs := Parse("Ralph item 3 (csv-serializer) completed")
	require.Len(t, refs, 1)
	assert.Equal(t, 3, refs[0].Index)
	assert.Equal(t, "csv-serializer", refs[0].Key)
}

func TestParseTrailerInsideCommitMessage(t *testing.T) {
	refs := Parse("feat: add csv serializer\n\nImplementation details.\n\nRalph item 3 (csv-serializer) completed")
	require.Len(t, refs, 1)
	assert.Equal(t, 3, refs[0].Index)
	assert.Equal(t, "csv-serializer", refs[0].Key)
}

func TestParseTrailerWithoutTrailingNewline(t *testing.T) {
	refs := Parse("feat: done\n\nRalph item 2 completed")
	require.Len(t, refs, 1)
	assert.Equal(t, 2, refs[0].Index)
}

func TestParseTrailerWithTrailingWhitespace(t *testing.T) {
	refs := Parse("Ralph item 1 completed   \n")
	require.Len(t, refs, 1)
	assert.Equal(t, 1, refs[0].Index)
}

func TestParseMultipleTrailers(t *testing.T) {
	refs := Parse("feat: export\n\nRalph item 1 completed\nRalph item 2 (export-endpoint) completed")
	require.Len(t, refs, 2)
	assert.Equal(t, 1, refs[0].Index)
	assert.Empty(t, refs[0].Key)
	assert.Equal(t, 2, refs[1].Index)
	assert.Equal(t, "export-endpoint", refs[1].Key)
}

func TestParseMessageWithoutTrailer(t *testing.T) {
	assert.Empty(t, Parse("feat: fix bug\n\nRefactor the parser."))
}

func TestParseIgnoresTrailerMentionedInProse(t *testing.T) {
	refs := Parse("README: the line `Ralph item 0 completed` marks an item done")
	assert.Empty(t, refs)
}

func TestParseIgnoresMalformedTrailers(t *testing.T) {
	for _, message := range []string{
		"Ralph item completed",
		"Ralph item -1 completed",
		"Ralph item 0",
		"completed Ralph item 0",
		"Ralph item 0 (completed",
		"Ralph item 0 () completed",
		"Ralph item 0 (a) b completed",
		"ralph item 0 completed",
		"Ralph item 2 (key) completed extra",
	} {
		assert.Empty(t, Parse(message), "message %q should not parse", message)
	}
}

func TestScenarioIndexOnlyTrailerRecognized(t *testing.T) {
	refs := Parse("feat: add serializer\n\nRalph item 0 completed")
	require.Len(t, refs, 1)
	assert.Equal(t, 0, refs[0].Index)
}

func TestScenarioTrailerWithKeyRecognized(t *testing.T) {
	refs := Parse("feat: export\n\nRalph item 3 (csv-serializer) completed")
	require.Len(t, refs, 1)
	assert.Equal(t, 3, refs[0].Index)
	assert.Equal(t, "csv-serializer", refs[0].Key)
}

func TestScenarioMultipleTrailersInOneCommit(t *testing.T) {
	refs := Parse("feat: everything\n\nRalph item 1 completed\nRalph item 2 (export-endpoint) completed")
	require.Len(t, refs, 2)
	indices := []int{refs[0].Index, refs[1].Index}
	assert.Equal(t, []int{1, 2}, indices)
	assert.Equal(t, "export-endpoint", refs[1].Key)
}

func TestScenarioCommitWithoutTrailerCompletesNothing(t *testing.T) {
	assert.Empty(t, Parse("feat: fix bug\n\nNo completion trailer in this message."))
}
