package run

import (
	"strings"
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
	prompt := buildChangelogPrompt("/tmp/report.md")

	assert.NotEmpty(t, prompt, "changelog prompt should not be empty")
	assert.Contains(t, prompt, "report.md", "prompt should reference report.md")
	assert.Contains(t, prompt, "git diff", "prompt should instruct inspecting git diff")
}

func TestBuildChangelogPromptTemplateUsage(t *testing.T) {
	prompt := buildChangelogPrompt("/tmp/report.md")

	expected := `Write a concise changelog entry for the changes currently staged in git.

You are an AI agent that writes changelogs. Review the git diff (staged changes) and write a single changelog entry describing what changed.

Focus on:
• What was added, removed, or modified
• Why the changes were made (if apparent from the diff)
• Any notable implementation details

Write in the style of a conventional changelog entry, beginning with a verb in past tense (e.g., "Fixed", "Added", "Changed").

Write the changelog entry to the file: /tmp/report.md

Do not include any extra commentary, just the changelog entry.
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(prompt), "changelog prompt should match expected template output")
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

func TestGenerateReviewPRBodyNoProject(t *testing.T) {
	ctx := testutil.NewContext()

	_, err := GenerateReviewPRBody(ctx, "nonexistent.yaml")
	assert.Error(t, err, "GenerateReviewPRBody should fail with nonexistent project file")
}

func TestBuildReviewPRBodyPrompt(t *testing.T) {
	prompt := buildReviewPRBodyPrompt(
		"review-2026-03-22",
		"Code review for authentication",
		[]string{"- **security**: JWT validation (✅ Passing)", "- **style**: naming conventions (❌ Not passing)"},
		"/tmp/pr-body.txt",
	)

	assert.NotEmpty(t, prompt, "PR body prompt should not be empty")
	assert.Contains(t, prompt, "review-2026-03-22", "prompt should include review name")
	assert.Contains(t, prompt, "Code review for authentication", "prompt should include description")
	assert.Contains(t, prompt, "JWT validation", "prompt should include requirement details")
	assert.Contains(t, prompt, "/tmp/pr-body.txt", "prompt should include output file path")
}

func TestBuildReviewPRBodyPromptNoDescription(t *testing.T) {
	prompt := buildReviewPRBodyPrompt(
		"review-2026-03-22",
		"",
		[]string{"- **security**: JWT validation (✅ Passing)"},
		"/tmp/pr-body.txt",
	)

	assert.NotEmpty(t, prompt, "PR body prompt should not be empty")
	assert.Contains(t, prompt, "review-2026-03-22", "prompt should include review name")
	assert.NotContains(t, prompt, "Description:", "prompt should not include empty description")
}
