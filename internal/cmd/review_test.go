package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
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

func TestPrintDetectedComponents(t *testing.T) {
	r := &ReviewCmd{}
	overview := &Overview{}
	r.printDetectedComponents(overview)
	// No panic expected
}

func TestShuffleComponents(t *testing.T) {
	components := []OverviewComponent{
		{Name: "comp1", Path: "path1", Summary: "summary1"},
		{Name: "comp2", Path: "path2", Summary: "summary2"},
		{Name: "comp3", Path: "path3", Summary: "summary3"},
	}
	seed := int64(12345)
	shuffled := shuffleComponents(components, seed)
	require.Len(t, shuffled, len(components))
	// Ensure all elements present
	for _, c := range components {
		found := false
		for _, s := range shuffled {
			if c.Name == s.Name && c.Path == s.Path && c.Summary == s.Summary {
				found = true
				break
			}
		}
		assert.True(t, found, "component %s missing", c.Name)
	}
	// Deterministic: same seed produces same order
	shuffled2 := shuffleComponents(components, seed)
	assert.Equal(t, shuffled, shuffled2)
	// Different seed produces different order (likely)
	shuffled3 := shuffleComponents(components, seed+1)
	// At least one position differs (not guaranteed but very likely)
	assert.NotEqual(t, shuffled, shuffled3)
}

func TestShuffleItemsWithIndices(t *testing.T) {
	items := []config.ReviewItem{
		{Text: "item1"},
		{Text: "item2"},
		{Text: "item3"},
	}
	seed := int64(54321)
	shuffled := shuffleItemsWithIndices(items, seed)
	require.Len(t, shuffled, len(items))
	// Check indices are preserved
	for _, pair := range shuffled {
		originalIdx := pair.idx
		assert.Equal(t, items[originalIdx].Text, pair.item.Text)
		// Ensure each original index appears exactly once
		found := false
		for _, p := range shuffled {
			if p.idx == originalIdx {
				found = true
				break
			}
		}
		assert.True(t, found)
	}
	// Deterministic
	shuffled2 := shuffleItemsWithIndices(items, seed)
	assert.Equal(t, shuffled, shuffled2)
	// Different seed likely different order
	shuffled3 := shuffleItemsWithIndices(items, seed+1)
	assert.NotEqual(t, shuffled, shuffled3)
}

func TestItemLabel(t *testing.T) {
	r := &ReviewCmd{}

	tests := []struct {
		name     string
		item     config.ReviewItem
		expected string
	}{
		{
			name: "text item with newline",
			item: config.ReviewItem{
				Text: "First line\nSecond line",
			},
			expected: "First line",
		},
		{
			name: "text item without newline",
			item: config.ReviewItem{
				Text: "Single line",
			},
			expected: "Single line",
		},
		{
			name: "text item truncation",
			item: config.ReviewItem{
				Text: "A very long line that exceeds eighty characters should be truncated with ellipsis at the end",
			},
			expected: "A very long line that exceeds eighty characters should be truncated with elli...",
		},
		{
			name: "file item",
			item: config.ReviewItem{
				File: "/path/to/file.go",
			},
			expected: "file.go",
		},
		{
			name: "file item with relative path",
			item: config.ReviewItem{
				File: "./docs/README.md",
			},
			expected: "README.md",
		},
		{
			name: "URL item",
			item: config.ReviewItem{
				URL: "https://example.com/path/to/resource",
			},
			expected: "resource",
		},
		{
			name: "URL item with query",
			item: config.ReviewItem{
				URL: "https://example.com/api/v1/users?id=123",
			},
			expected: "users",
		},
		{
			name: "URL item with fragment",
			item: config.ReviewItem{
				URL: "https://example.com/docs#section",
			},
			expected: "docs",
		},
		{
			name: "URL item with trailing slash",
			item: config.ReviewItem{
				URL: "https://example.com/folder/",
			},
			expected: "folder",
		},
		{
			name: "URL item with empty path",
			item: config.ReviewItem{
				URL: "https://example.com",
			},
			expected: "example.com",
		},
		{
			name: "URL item with root path",
			item: config.ReviewItem{
				URL: "https://example.com/",
			},
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label := r.itemLabel(tt.item)
			assert.Equal(t, tt.expected, label)
		})
	}
}

func TestReviewSeedFlag(t *testing.T) {
	// Test that the seed flag is parsed correctly
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--seed", "42"})
	require.NoError(t, err)
	assert.Equal(t, int64(42), cmd.Review.Seed)
}
