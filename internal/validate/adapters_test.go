package validate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractYAML_RemovesYAMLFence(t *testing.T) {
	response := "```yaml\nslug: test\nrequirements:\n  - slug: req\n    passing: true\n```"
	result := extractYAML(response)
	require.Equal(t, "slug: test\nrequirements:\n  - slug: req\n    passing: true", string(result))
}

func TestExtractYAML_RemovesPlainFence(t *testing.T) {
	response := "```\nslug: test\npassing: true\n```"
	result := extractYAML(response)
	require.Equal(t, "slug: test\npassing: true", string(result))
}

func TestExtractYAML_NoFence(t *testing.T) {
	response := "slug: test\npassing: true"
	result := extractYAML(response)
	require.Equal(t, "slug: test\npassing: true", string(result))
}

func TestExtractYAML_TrimsWhitespace(t *testing.T) {
	response := "  \nslug: test\npassing: true\n  "
	result := extractYAML(response)
	require.Equal(t, "slug: test\npassing: true", string(result))
}

func TestExtractYAML_EmptyString(t *testing.T) {
	response := ""
	result := extractYAML(response)
	require.Empty(t, result)
}

func TestExtractYAML_EmptyWithWhitespace(t *testing.T) {
	response := "  \n  \n  "
	result := extractYAML(response)
	require.Empty(t, result)
}

func TestExtractYAML_YAMLFenceWithLeadingNewline(t *testing.T) {
	response := "\n\n```yaml\nkey: value\n```\n\n"
	result := extractYAML(response)
	require.Equal(t, "key: value", string(result))
}

func TestExtractYAML_PlainFenceWithLeadingNewline(t *testing.T) {
	response := "\n```\nkey: value\n```\n"
	result := extractYAML(response)
	require.Equal(t, "key: value", string(result))
}

func TestExtractYAML_NoNewlineBeforeClose(t *testing.T) {
	response := "```yaml\nkey: value```"
	result := extractYAML(response)
	require.Equal(t, "key: value", string(result))
}

func TestExtractYAML_EmptyFenceIsKept(t *testing.T) {
	response := "``````"
	result := extractYAML(response)
	require.Equal(t, "``````", string(result))
}
