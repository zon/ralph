package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
)

func TestReviewResolveModel(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `model: deepseek/deepseek-chat
review:
  model: google/gemini-2.5-pro
  items:
  - text: test review
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	r := &ReviewCmd{
		Model: "",
	}

	model := r.resolveModel(cfg)
	assert.Equal(t, "google/gemini-2.5-pro", model)
}

func TestReviewResolveModel_WithFlagOverride(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `model: deepseek/deepseek-chat
review:
  model: google/gemini-2.5-pro
  items:
  - text: test review
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	r := &ReviewCmd{
		Model: "anthropic/claude-3-sonnet",
	}

	model := r.resolveModel(cfg)
	assert.Equal(t, "anthropic/claude-3-sonnet", model)
}

func TestReviewResolveModel_FallbackToMainModel(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configContent := `model: deepseek/deepseek-chat
review:
  items:
  - text: test review
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	t.Chdir(tmpDir)

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	r := &ReviewCmd{
		Model: "",
	}

	model := r.resolveModel(cfg)
	assert.Equal(t, "deepseek/deepseek-chat", model)
}

func TestReviewLoadItemContent_Text(t *testing.T) {
	r := &ReviewCmd{}
	item := config.ReviewItem{
		Text: "inline text content",
	}

	content, err := r.loadItemContent(item)
	require.NoError(t, err)
	assert.Equal(t, "inline text content", content)
}

func TestReviewLoadItemContent_File(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte("file content"), 0644))

	r := &ReviewCmd{}
	item := config.ReviewItem{
		File: testFile,
	}

	content, err := r.loadItemContent(item)
	require.NoError(t, err)
	assert.Equal(t, "file content", content)
}

func TestReviewLoadItemContent_FileNotFound(t *testing.T) {
	r := &ReviewCmd{}
	item := config.ReviewItem{
		File: "/nonexistent/file.md",
	}

	_, err := r.loadItemContent(item)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestReviewBuildPrompt(t *testing.T) {
	r := &ReviewCmd{}
	prompt := r.buildPrompt("test content", "/path/to/project.yaml", "project doc content", "review-2026-03-22")

	assert.Contains(t, prompt, "You are a software architect reviewing source code")
	assert.Contains(t, prompt, "test content")
	assert.Contains(t, prompt, "/path/to/project.yaml")
	assert.Contains(t, prompt, "project doc content")
	assert.Contains(t, prompt, "review-2026-03-22")
}

func TestEmbeddedReviewInstructions(t *testing.T) {
	if reviewInstructions == "" {
		t.Error("reviewInstructions should not be empty")
	}
	if !contains(reviewInstructions, "{{.ConfigContent}}") {
		t.Error("reviewInstructions should contain ConfigContent template variable")
	}
	if !contains(reviewInstructions, "{{.ReviewName}}") {
		t.Error("reviewInstructions should contain ReviewName template variable")
	}
}

func TestReviewBuildCommitMessage(t *testing.T) {
	tests := []struct {
		name         string
		component    string
		itemIndex    int
		summaryPath  string
		summary      string
		wantContains string
	}{
		{
			name:         "commit message with summary",
			component:    "internal-git",
			itemIndex:    0,
			summaryPath:  "tmp/summary.txt",
			summary:      "Added validation for commit message format",
			wantContains: "review: internal-git-0 Added validation for commit message format",
		},
		{
			name:         "commit message without summary file",
			component:    "internal-api",
			itemIndex:    1,
			summaryPath:  "/nonexistent/summary.txt",
			summary:      "",
			wantContains: "review: internal-api-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.summary != "" {
				tmpDir := t.TempDir()
				tt.summaryPath = filepath.Join(tmpDir, "summary.txt")
				require.NoError(t, os.WriteFile(tt.summaryPath, []byte(tt.summary), 0644))
			}

			r := &ReviewCmd{}
			msg := r.buildCommitMessage(tt.component, tt.itemIndex, tt.summaryPath)

			assert.Equal(t, tt.wantContains, msg)
		})
	}
}

// TestReviewCommitPrefixConsistency verifies that the prefix used in commit messages
// matches the prefix searched for in the git log when checking for skips.
// The format must be '$component-$item' (0-indexed) for both operations.
func TestReviewCommitPrefixConsistency(t *testing.T) {
	tests := []struct {
		component      string
		itemIndex      int
		expectedPrefix string
	}{
		{"internal-git", 0, "internal-git-0"},
		{"internal-api", 1, "internal-api-1"},
		{"cmd", 2, "cmd-2"},
		{"webhook", 0, "webhook-0"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedPrefix, func(t *testing.T) {
			r := &ReviewCmd{}

			// Commit message must contain the deterministic prefix
			tmpDir := t.TempDir()
			summaryPath := filepath.Join(tmpDir, "summary.txt")
			require.NoError(t, os.WriteFile(summaryPath, []byte("some summary"), 0644))

			msg := r.buildCommitMessage(tt.component, tt.itemIndex, summaryPath)

			// The message must contain the $component-$item prefix
			assert.Contains(t, msg, tt.expectedPrefix,
				"commit message must contain the $component-$item prefix")

			// The prefix must appear near the start of the message (after optional "review: ")
			assert.Contains(t, msg[:len("review: ")+len(tt.expectedPrefix)], tt.expectedPrefix,
				"$component-$item prefix must appear at the start of the commit message")
		})
	}
}

func TestReviewIterationPrefixFormat(t *testing.T) {
	// Verify that the iteration prefix format '$component-$item' is deterministic
	// and 0-indexed as required by the specification.
	tests := []struct {
		component string
		itemIndex int
		want      string
	}{
		{"internal-git", 0, "internal-git-0"},
		{"internal-git", 1, "internal-git-1"},
		{"internal-git", 9, "internal-git-9"},
		{"internal-api", 0, "internal-api-0"},
		{"cmd", 5, "cmd-5"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			r := &ReviewCmd{}
			// Build a commit message with no summary to isolate prefix
			msg := r.buildCommitMessage(tt.component, tt.itemIndex, "/nonexistent/path")
			// The prefix must be in the message in the $component-$item format
			assert.Contains(t, msg, tt.want)
		})
	}
}
