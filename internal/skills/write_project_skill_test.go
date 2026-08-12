package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func readWriteProjectSkill(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", ".claude", "skills", "ralph-write-project", "SKILL.md")
	content, err := os.ReadFile(path)
	require.NoError(t, err, "skill should exist at %s", path)
	return string(content)
}

func TestWriteProjectSkillDescribesItemArrayFormat(t *testing.T) {
	skill := readWriteProjectSkill(t)

	require.Contains(t, skill, "any YAML or JSON file")
	require.Contains(t, skill, "array of work items")
	require.Contains(t, skill, "jq")
	require.Contains(t, skill, "item query")
}

func TestWriteProjectSkillExplainsIndexKeyAndNoCompletionField(t *testing.T) {
	skill := readWriteProjectSkill(t)

	require.Contains(t, skill, "index")
	require.Contains(t, skill, "slug")
	require.Contains(t, skill, "id")
	require.Contains(t, skill, "name")
	require.Contains(t, skill, "key")
	require.Contains(t, skill, "No completion field belongs in the file")
}

func TestWriteProjectSkillInstructsConventionalShapeAndKeys(t *testing.T) {
	skill := readWriteProjectSkill(t)

	require.Contains(t, skill, "conventional")
	require.Contains(t, skill, "give each item a key")
	require.Contains(t, skill, "commit trailers")
}

func TestWriteProjectSkillDoesNotInstructPassingField(t *testing.T) {
	skill := readWriteProjectSkill(t)

	require.NotContains(t, skill, "passing")
}

func TestWriteProjectSkillStatesSlugAndTitleFallbacks(t *testing.T) {
	skill := readWriteProjectSkill(t)

	require.Contains(t, skill, "slug")
	require.Contains(t, skill, "title")
	require.Contains(t, skill, "optional")
	require.Contains(t, skill, "fall back")
	require.Contains(t, skill, "file name")
}

func TestWriteProjectSkillRepoFilesAreMarkdownLinks(t *testing.T) {
	skill := readWriteProjectSkill(t)
	rewritten := rewriteLinks(skill, "main")

	referenced := []string{
		"docs/formats/project.md",
		"docs/code.md",
		"docs/testing.md",
		"docs/formats/architecture.md",
		"specs/architecture.yaml",
	}
	for _, f := range referenced {
		require.Contains(t, rewritten, "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/"+f,
			"file %s should be referenced as a markdown link rewritable to a raw URL", f)
	}
}
