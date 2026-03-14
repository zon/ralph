package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zon/ralph/internal/testutil"
)

func TestGeneratePRSummaryNoProject(t *testing.T) {
	ctx := testutil.NewContext()

	_, err := GeneratePRSummary(ctx, "nonexistent.yaml", 1, "main")
	assert.Error(t, err, "GeneratePRSummary should fail with nonexistent project file")
}

func TestBuildChangelogPrompt(t *testing.T) {
	prompt := buildChangelogPrompt()

	assert.NotEmpty(t, prompt, "changelog prompt should not be empty")
	assert.Contains(t, prompt, "report.md", "prompt should reference report.md")
	assert.Contains(t, prompt, "git diff", "prompt should instruct inspecting git diff")
}

func TestBuildChangelogPromptTemplateUsage(t *testing.T) {
	prompt := buildChangelogPrompt()

	expected := `Inspect the current uncommitted git changes and write a concise changelog entry to 'report.md'.

Steps:
1. Run 'git diff HEAD' (or 'git diff --cached' for staged-only) to see what changed.
2. Write a short, commit-message-style summary of the changes to 'report.md'.

Format:
- First line: imperative-mood summary (≤72 chars), e.g. "Add user authentication endpoint"
- Optional blank line followed by bullet points for non-obvious details

Keep it concise — this becomes the git commit message.
Do NOT include code snippets or verbose explanations.
`
	assert.Equal(t, expected, prompt, "changelog prompt should match expected template output")
}

func TestBuildPRSummaryPrompt(t *testing.T) {
	prompt := buildPRSummaryPrompt(
		"Test Project",
		"✅ Complete",
		"main",
		"abc123: Initial commit\ndef456: Add feature\n",
		"/tmp/pr-summary.txt",
	)

	assert.NotEmpty(t, prompt, "PR summary prompt should not be empty")
	assert.Contains(t, prompt, "Test Project", "prompt should include project description")
	assert.Contains(t, prompt, "✅ Complete", "prompt should include project status")
	assert.Contains(t, prompt, "main..HEAD", "prompt should reference base branch")
	assert.Contains(t, prompt, "abc123: Initial commit", "prompt should include commit log")
	assert.Contains(t, prompt, "/tmp/pr-summary.txt", "prompt should include output file path")
}
