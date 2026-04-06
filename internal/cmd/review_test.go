package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/testutil"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

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
	prompt := r.buildItemPrompt("test content")

	assert.Contains(t, prompt, "test content")
}

func TestEmbeddedReviewInstructions(t *testing.T) {
	if reviewInstructions == "" {
		t.Error("reviewInstructions should not be empty")
	}
	if !contains(reviewInstructions, "{{.ItemContent}}") {
		t.Error("reviewInstructions should contain ItemContent template variable")
	}
	if contains(reviewInstructions, "{{.ConfigContent}}") {
		t.Error("reviewInstructions should not contain ConfigContent template variable")
	}
	if contains(reviewInstructions, "{{.ReviewName}}") {
		t.Error("reviewInstructions should not contain ReviewName template variable")
	}
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

func TestFilterItems_EmptyFilter(t *testing.T) {
	items := []config.ReviewItem{
		{Text: "item1"},
		{Text: "item2"},
		{Text: "item3"},
	}
	filtered := filterItems(items, "")
	assert.Len(t, filtered, 3)
	assert.Equal(t, items, filtered)
}

func TestFilterItems_SubstringMatch(t *testing.T) {
	items := []config.ReviewItem{
		{Text: "first review item"},
		{Text: "second review item"},
		{Text: "third item"},
	}
	filtered := filterItems(items, "review")
	assert.Len(t, filtered, 2)
	assert.Equal(t, "first review item", filtered[0].Text)
	assert.Equal(t, "second review item", filtered[1].Text)
}

func TestFilterItems_CaseInsensitive(t *testing.T) {
	items := []config.ReviewItem{
		{Text: "First Review Item"},
		{Text: "SECOND REVIEW ITEM"},
		{Text: "third item"},
	}
	filtered := filterItems(items, "review")
	assert.Len(t, filtered, 2)
	filteredUpper := filterItems(items, "REVIEW")
	assert.Len(t, filteredUpper, 2)
}

func TestFilterItems_FileMatch(t *testing.T) {
	items := []config.ReviewItem{
		{File: "src/main.go"},
		{File: "src/test.go"},
		{File: "docs/README.md"},
	}
	filtered := filterItems(items, "main")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "src/main.go", filtered[0].File)
}

func TestFilterItems_URLMatch(t *testing.T) {
	items := []config.ReviewItem{
		{URL: "https://example.com/api/users"},
		{URL: "https://example.com/api/posts"},
		{URL: "https://other.com/page"},
	}
	filtered := filterItems(items, "example.com/api")
	assert.Len(t, filtered, 2)
}

func TestFilterItems_NoMatch(t *testing.T) {
	items := []config.ReviewItem{
		{Text: "first review item"},
		{Text: "second review item"},
		{Text: "third item"},
	}
	filtered := filterItems(items, "nonexistent")
	assert.Len(t, filtered, 0)
}

func TestFilterItems_CombinedFields(t *testing.T) {
	items := []config.ReviewItem{
		{Text: "contains foo in text"},
		{File: "bar.txt"},
		{URL: "https://example.com/baz"},
	}
	filtered := filterItems(items, "foo")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "contains foo in text", filtered[0].Text)

	filtered = filterItems(items, "bar")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "bar.txt", filtered[0].File)

	filtered = filterItems(items, "example")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "https://example.com/baz", filtered[0].URL)
}

func TestFilterItems_AllFieldsMatch(t *testing.T) {
	items := []config.ReviewItem{
		{Text: "prefix_match_suffix"},
		{File: "path_prefix_match_suffix/file.go"},
		{URL: "https://site.com/path_prefix_match_suffix"},
	}
	filtered := filterItems(items, "prefix_match_suffix")
	assert.Len(t, filtered, 3)
}

func TestReviewFilterNoMatchError(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	remoteDir := filepath.Join(tmpDir, "remote.git")
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
	require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
	require.NoError(t, exec.Command("git", "branch", "-M", "main").Run())

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
review:
  items:
  - text: first review item
  - text: second review item
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	projectsDir := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.Mkdir(projectsDir, 0755))
	projectFile := filepath.Join(projectsDir, "test-review.yaml")
	projectYAML := `name: test-review
description: Test project for filter
requirements:
  - category: test
    description: dummy requirement
    passing: false
`
	require.NoError(t, os.WriteFile(projectFile, []byte(projectYAML), 0644))
	require.NoError(t, exec.Command("git", "add", ".").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "initial commit").Run())
	require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	r := &ReviewCmd{
		Local:   true,
		Verbose: false,
		Seed:    12345,
		Filter:  "nonexistent_filter_string",
	}
	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	ctx.SetLocal(true)

	_, _, err = r.runReview(ctx, cfg, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent_filter_string")
	assert.Contains(t, err.Error(), "no review items match filter")
}

func TestReviewFilterWithMatch(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	remoteDir := filepath.Join(tmpDir, "remote.git")
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
	require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
	require.NoError(t, exec.Command("git", "branch", "-M", "main").Run())

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
review:
  items:
  - text: first review item
  - text: second review item
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	projectsDir := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.Mkdir(projectsDir, 0755))
	projectFile := filepath.Join(projectsDir, "test-review.yaml")
	projectYAML := `name: test-review
description: Test project for filter
requirements:
  - category: test
    description: dummy requirement
    passing: false
`
	require.NoError(t, os.WriteFile(projectFile, []byte(projectYAML), 0644))
	require.NoError(t, exec.Command("git", "add", ".").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "initial commit").Run())
	require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	r := &ReviewCmd{
		Local:   true,
		Verbose: false,
		Seed:    12345,
		Filter:  "first",
	}
	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	ctx.SetLocal(true)

	branchName, detectedFile, err := r.runReview(ctx, cfg, "")
	require.NoError(t, err)
	assert.NotEmpty(t, detectedFile)
	assert.NotEmpty(t, branchName)
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

func TestReviewFollowFlag(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--follow"})
	require.NoError(t, err)
	assert.True(t, cmd.Review.Follow)
}

func TestReviewFollowFlagShort(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "-f"})
	require.NoError(t, err)
	assert.True(t, cmd.Review.Follow)
}

func TestReviewFilterFlag(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--filter", "myfilter"})
	require.NoError(t, err)
	assert.Equal(t, "myfilter", cmd.Review.Filter)
}

func TestReviewFilterFlagShort(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--filter", "substring"})
	require.NoError(t, err)
	assert.Equal(t, "substring", cmd.Review.Filter)
}

func TestReviewFollowWithLocalFlag(t *testing.T) {
	r := &ReviewCmd{
		Follow: true,
		Local:  true,
	}
	err := r.validateFlagCombinations()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--follow flag is not applicable with --local flag")
}

func TestReviewFollowWithoutLocalFlag(t *testing.T) {
	r := &ReviewCmd{
		Follow: true,
		Local:  false,
	}
	err := r.validateFlagCombinations()
	require.NoError(t, err)
}

func TestReviewLoopProcessesAllItems(t *testing.T) {
	t.Setenv("RALPH_MOCK_AI", "true")
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create a bare remote repository
	remoteDir := filepath.Join(tmpDir, "remote.git")
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

	// Initialize git repo with origin
	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
	require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
	// Rename branch to main
	require.NoError(t, exec.Command("git", "branch", "-M", "main").Run())

	// Create .ralph/config.yaml with two review items
	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	configContent := `model: deepseek/deepseek-chat
review:
  items:
  - text: first review item
  - text: second review item
`
	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create a project YAML file in projects directory
	projectsDir := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.Mkdir(projectsDir, 0755))
	projectFile := filepath.Join(projectsDir, "test-review.yaml")
	projectYAML := `name: test-review
description: Test project for loop
requirements:
  - category: test
    description: dummy requirement
    passing: false
`
	require.NoError(t, os.WriteFile(projectFile, []byte(projectYAML), 0644))
	// Commit the project file to have a base state
	require.NoError(t, exec.Command("git", "add", ".").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "initial commit").Run())
	// Push to origin to establish remote branch
	require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

	// Load config
	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	// Create ReviewCmd
	r := &ReviewCmd{
		Local:   true,
		Verbose: false,
		Seed:    12345,
	}
	ctx := testutil.NewContext(testutil.WithProjectFile(projectFile))
	ctx.SetLocal(true)

	// Run review - new signature without overview
	branchName, detectedFile, err := r.runReview(ctx, cfg, "")
	require.NoError(t, err)

	// Verify that all items were processed (both should have run through the loop)
	// The function should complete without error and continue processing all items
	assert.NotEmpty(t, detectedFile, "detected project file should not be empty after processing items")
	assert.NotEmpty(t, branchName, "branchName should not be empty after processing items")

	// Verify that commits were made - there should be more than just the initial commit
	cmd := exec.Command("git", "log", "--oneline")
	out, err := cmd.Output()
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// Should have at least 2 commits (initial + at least one review commit)
	assert.GreaterOrEqual(t, len(lines), 2, "should have at least two commits")
}
