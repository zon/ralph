package ai

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestCaptureWriterTail(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		buf      string
		expected string
	}{
		{
			name:     "empty",
			lines:    []string{},
			buf:      "",
			expected: "",
		},
		{
			name:     "fewer than n lines",
			lines:    []string{"line1", "line2", "line3"},
			buf:      "",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "with partial line",
			lines:    []string{"line1", "line2"},
			buf:      "line3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "exactly n lines",
			lines:    []string{"line1", "line2", "line3"},
			buf:      "",
			expected: "line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cw := &ringWriter{n: 10, lines: tt.lines, buf: tt.buf}
			result := cw.tail()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRunAgentErrorIncludesTail(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake-opencode.sh")

	scriptContent := `#!/bin/bash
echo "line 1 output"
echo "line 2 output"
echo "line 3 output"
echo "line 4 output"
echo "line 5 output"
echo "line 6 output"
echo "line 7 output"
echo "line 8 output"
echo "line 9 output"
echo "line 10 output"
echo "line 11 output"
echo "line 12 output"
exit 1
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	opencodePath := filepath.Join(tmpDir, "opencode")
	err = os.Symlink(scriptPath, opencodePath)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+origPath)
	t.Setenv("RALPH_MOCK_AI", "")

	ctx := testutil.NewContext()
	err = RunAgent(ctx, "test prompt")

	require.Error(t, err, "RunAgent should return error when opencode fails")
	assert.Contains(t, err.Error(), "opencode execution failed")
	assert.Contains(t, err.Error(), "line 3")
	assert.Contains(t, err.Error(), "line 12")
	assert.NotContains(t, err.Error(), "line 2 output", "Should not include lines before last 10")
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
