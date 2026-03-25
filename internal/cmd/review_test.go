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
