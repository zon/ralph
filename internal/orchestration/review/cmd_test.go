package review

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
)

// ---------------------------------------------------------------------------
// Pure function tests: shuffleItemsWithIndices
// ---------------------------------------------------------------------------

func TestShuffleItemsWithIndices(t *testing.T) {
	items := []config.ReviewItem{
		{Text: "item1"},
		{Text: "item2"},
		{Text: "item3"},
	}
	seed := int64(54321)
	shuffled := shuffleItemsWithIndices(items, seed)
	require.Len(t, shuffled, len(items))
	for _, pair := range shuffled {
		originalIdx := pair.idx
		assert.Equal(t, items[originalIdx].Text, pair.item.Text)
		found := false
		for _, p := range shuffled {
			if p.idx == originalIdx {
				found = true
				break
			}
		}
		assert.True(t, found)
	}
	shuffled2 := shuffleItemsWithIndices(items, seed)
	assert.Equal(t, shuffled, shuffled2)
	shuffled3 := shuffleItemsWithIndices(items, seed+1)
	assert.NotEqual(t, shuffled, shuffled3)
}

// ---------------------------------------------------------------------------
// Pure function tests: filterItems
// ---------------------------------------------------------------------------

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

func TestFilterItems_LoopMatch(t *testing.T) {
	items := []config.ReviewItem{
		{Text: "regular item"},
		{Loop: "domain-function"},
		{Text: "another item", Loop: "domain-function"},
	}
	filtered := filterItems(items, "domain-function")
	assert.Len(t, filtered, 2)
	assert.Equal(t, "domain-function", filtered[0].Loop)
	assert.Equal(t, "domain-function", filtered[1].Loop)
}

func TestFilterItems_LoopNoMatch(t *testing.T) {
	items := []config.ReviewItem{
		{Loop: "domain-function"},
		{Loop: "other-loop"},
	}
	filtered := filterItems(items, "nonexistent")
	assert.Len(t, filtered, 0)
}

// ---------------------------------------------------------------------------
// Pure function tests: itemLabel
// ---------------------------------------------------------------------------

func TestItemLabel(t *testing.T) {
	tests := []struct {
		name     string
		item     config.ReviewItem
		expected string
	}{
		{
			name:     "text item with newline",
			item:     config.ReviewItem{Text: "First line\nSecond line"},
			expected: "First line",
		},
		{
			name:     "text item without newline",
			item:     config.ReviewItem{Text: "Single line"},
			expected: "Single line",
		},
		{
			name:     "text item truncation",
			item:     config.ReviewItem{Text: "A very long line that exceeds eighty characters should be truncated with ellipsis at the end"},
			expected: "A very long line that exceeds eighty characters should be truncated with elli...",
		},
		{
			name:     "file item",
			item:     config.ReviewItem{File: "/path/to/file.go"},
			expected: "file.go",
		},
		{
			name:     "file item with relative path",
			item:     config.ReviewItem{File: "./docs/README.md"},
			expected: "README.md",
		},
		{
			name:     "URL item",
			item:     config.ReviewItem{URL: "https://example.com/path/to/resource"},
			expected: "resource",
		},
		{
			name:     "URL item with query",
			item:     config.ReviewItem{URL: "https://example.com/api/v1/users?id=123"},
			expected: "users",
		},
		{
			name:     "URL item with fragment",
			item:     config.ReviewItem{URL: "https://example.com/docs#section"},
			expected: "docs",
		},
		{
			name:     "URL item with trailing slash",
			item:     config.ReviewItem{URL: "https://example.com/folder/"},
			expected: "folder",
		},
		{
			name:     "URL item with empty path",
			item:     config.ReviewItem{URL: "https://example.com"},
			expected: "example.com",
		},
		{
			name:     "loop item with domain-function",
			item:     config.ReviewItem{Loop: "domain-function"},
			expected: "loop:domain-function",
		},
		{
			name:     "loop item takes precedence over text",
			item:     config.ReviewItem{Text: "Some text content", Loop: "domain-function"},
			expected: "loop:domain-function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label := itemLabel(tt.item)
			assert.Equal(t, tt.expected, label)
		})
	}
}

// ---------------------------------------------------------------------------
// Pure function tests: resolveBaseBranch
// ---------------------------------------------------------------------------

func TestResolveBaseBranch_FlagOverrides(t *testing.T) {
	result := resolveBaseBranch("develop", "feature-x", "my-project", "main")
	assert.Equal(t, "develop", result)
}

func TestResolveBaseBranch_CurrentBranchWhenDifferentFromProject(t *testing.T) {
	result := resolveBaseBranch("", "feature-x", "my-project", "main")
	assert.Equal(t, "feature-x", result)
}

func TestResolveBaseBranch_DefaultWhenOnProjectBranch(t *testing.T) {
	result := resolveBaseBranch("", "my-project", "my-project", "main")
	assert.Equal(t, "main", result)
}

// ---------------------------------------------------------------------------
// Model resolution tests
// ---------------------------------------------------------------------------

func TestResolveModel_ReviewModelFromConfig(t *testing.T) {
	cmd := withMocks()
	cfg := configWithReviewModel("google/gemini-2.5-pro")
	flags := ReviewFlags{}
	model := cmd.resolveModel(flags, cfg)
	assert.Equal(t, "google/gemini-2.5-pro", model)
}

func TestResolveModel_FlagOverride(t *testing.T) {
	cmd := withMocks()
	cfg := configWithReviewModel("google/gemini-2.5-pro")
	flags := ReviewFlags{Model: "anthropic/claude-3-sonnet"}
	model := cmd.resolveModel(flags, cfg)
	assert.Equal(t, "anthropic/claude-3-sonnet", model)
}

func TestResolveModel_FallbackToMainModel(t *testing.T) {
	cmd := withMocks()
	cfg := configWithModel("deepseek/deepseek-chat")
	flags := ReviewFlags{}
	model := cmd.resolveModel(flags, cfg)
	assert.Equal(t, "deepseek/deepseek-chat", model)
}

// ---------------------------------------------------------------------------
// Item content loading tests
// ---------------------------------------------------------------------------

func TestLoadItemContent_Text(t *testing.T) {
	cmd := withMocks()
	item := config.ReviewItem{Text: "inline text content"}
	content, err := cmd.loadItemContent(item)
	require.NoError(t, err)
	assert.Equal(t, "inline text content", content)
}

func TestLoadItemContent_File(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte("file content"), 0644))

	cmd := withMocks()
	item := config.ReviewItem{File: testFile}
	content, err := cmd.loadItemContent(item)
	require.NoError(t, err)
	assert.Equal(t, "file content", content)
}

func TestLoadItemContent_FileNotFound(t *testing.T) {
	cmd := withMocks()
	item := config.ReviewItem{File: "/nonexistent/file.md"}
	_, err := cmd.loadItemContent(item)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

// ---------------------------------------------------------------------------
// Flag validation tests
// ---------------------------------------------------------------------------

func TestReviewFollowWithLocalFlag(t *testing.T) {
	flags := ReviewFlags{Follow: true, Local: true}
	err := flags.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--follow flag is not applicable with --local flag")
}

func TestReviewFollowWithoutLocalFlag(t *testing.T) {
	flags := ReviewFlags{Follow: true, Local: false}
	err := flags.Validate()
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// runReview tests (with mocks)
// ---------------------------------------------------------------------------

func TestRunReview_FilterNoMatchReturnsError(t *testing.T) {
	mockAI := &mockAIClient{}
	cmd := withMocks(withAI(mockAI))

	flags := ReviewFlags{Local: true, Filter: "nonexistent_filter_string"}
	cfg := configWithItems([]config.ReviewItem{
		{Text: "first review item"},
		{Text: "second review item"},
	})

	_, _, err := cmd.runReview(flags, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent_filter_string")
	assert.Contains(t, err.Error(), "no review items match filter")
}

func TestRunReview_FilterWithMatch(t *testing.T) {
	mockGit := &mockGitClient{
		hasUncommittedChangesFunc: func() bool { return false },
	}
	mockAI := &mockAIClient{}
	cmd := withMocks(withAI(mockAI), withGit(mockGit))

	flags := ReviewFlags{Local: true, Seed: 12345, Filter: "first"}
	cfg := configWithItems([]config.ReviewItem{
		{Text: "first review item"},
		{Text: "second review item"},
	})

	branchName, detectedFile, err := cmd.runReview(flags, cfg)
	require.NoError(t, err)
	assert.Empty(t, detectedFile)
	assert.Empty(t, branchName)
}

func TestRunReview_OneRunsSingleItem(t *testing.T) {
	mockAI := &mockAIClient{
		buildReviewItemPromptFunc: func(content string) (string, error) {
			return "prompt-" + content, nil
		},
	}
	cmd := withMocks(withAI(mockAI))

	flags := ReviewFlags{Local: true, Seed: 12345, One: true}
	cfg := configWithItems([]config.ReviewItem{
		{Text: "first review item"},
		{Text: "second review item"},
		{Text: "third review item"},
	})

	_, _, err := cmd.runReview(flags, cfg)
	require.NoError(t, err)

	calls := aiRunAgentCalls(cmd)
	assert.Len(t, calls, 1, "should have run exactly one item when --one is set")
}

func TestRunReview_RunsAllItems(t *testing.T) {
	mockAI := &mockAIClient{}
	cmd := withMocks(withAI(mockAI))

	flags := ReviewFlags{Local: true, Seed: 12345}
	cfg := configWithItems([]config.ReviewItem{
		{Text: "first review item"},
		{Text: "second review item"},
	})

	_, _, err := cmd.runReview(flags, cfg)
	require.NoError(t, err)

	calls := aiRunAgentCalls(cmd)
	assert.Len(t, calls, 2, "should have run all items")
}

// ---------------------------------------------------------------------------
// commitReviewItemChanges tests (with mocks)
// ---------------------------------------------------------------------------

func TestCommitReviewItemChanges_NoChanges(t *testing.T) {
	mockGit := &mockGitClient{
		tmpPathFunc: func(filename string) (string, error) {
			return filepath.Join(t.TempDir(), filename), nil
		},
	}
	cmd := withMocks(withGit(mockGit))

	err := cmd.commitReviewItemChanges("main", 0)
	require.NoError(t, err)

	calls := gitCommitAllAndPushCalls(cmd)
	require.Len(t, calls, 1)
	assert.Equal(t, "main", calls[0].branch)
}

// ---------------------------------------------------------------------------
// submitToArgo tests (with mocks)
// ---------------------------------------------------------------------------

func TestSubmitToArgo(t *testing.T) {
	mockWf := &mockWorkflowClient{}
	mockGit := &mockGitClient{
		isBranchSyncedWithRemoteFunc: func(branch string) error { return nil },
	}
	cmd := withMocks(withGit(mockGit), withWorkflow(mockWf))

	flags := ReviewFlags{Follow: false}
	err := cmd.submitToArgo(flags, "main")
	require.NoError(t, err)

	calls := workflowSubmitReviewCalls(cmd)
	require.Len(t, calls, 1)
	assert.Equal(t, "main", calls[0])
	assert.False(t, workflowFollowLogsCalled(cmd))
}

func TestSubmitToArgo_FollowLogs(t *testing.T) {
	mockWf := &mockWorkflowClient{}
	mockGit := &mockGitClient{
		isBranchSyncedWithRemoteFunc: func(branch string) error { return nil },
	}
	cmd := withMocks(withGit(mockGit), withWorkflow(mockWf))

	flags := ReviewFlags{Follow: true}
	err := cmd.submitToArgo(flags, "main")
	require.NoError(t, err)

	calls := workflowSubmitReviewCalls(cmd)
	require.Len(t, calls, 1)
	assert.True(t, workflowFollowLogsCalled(cmd))
}

func TestSubmitToArgo_BranchNotSynced(t *testing.T) {
	mockGit := &mockGitClient{
		isBranchSyncedWithRemoteFunc: func(branch string) error {
			return errMock
		},
	}
	cmd := withMocks(withGit(mockGit))

	err := cmd.submitToArgo(ReviewFlags{}, "main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mock error")
}

// ---------------------------------------------------------------------------
// Run dispatch tests
// ---------------------------------------------------------------------------

func TestRun_LocalDispatchesToRunReview(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))
	configContent := "model: deepseek/deepseek-chat\nreview:\n  items:\n  - text: test review item\n"
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte(configContent), 0644))

	require.NoError(t, os.MkdirAll("projects", 0755))

	mockAI := &mockAIClient{}
	mockGit := &mockGitClient{
		currentBranchFunc: func() (string, error) { return "feature-x", nil },
	}
	cmd := withMocks(withAI(mockAI), withGit(mockGit))

	flags := ReviewFlags{Local: true}
	err := cmd.Run(flags)
	require.NoError(t, err)
	assert.True(t, gitCurrentBranchCalled(cmd))
	assert.True(t, aiDisplayStatsCalled(cmd))
}

func TestRun_RemoteDispatchesToSubmitToArgo(t *testing.T) {
	mockGit := &mockGitClient{
		currentBranchFunc: func() (string, error) { return "main", nil },
		isBranchSyncedWithRemoteFunc: func(branch string) error { return nil },
	}
	mockWf := &mockWorkflowClient{}
	cmd := withMocks(withGit(mockGit), withWorkflow(mockWf))

	flags := ReviewFlags{Local: false}
	err := cmd.Run(flags)
	require.NoError(t, err)
	calls := workflowSubmitReviewCalls(cmd)
	require.Len(t, calls, 1)
	assert.Equal(t, "main", calls[0])
}

func TestRun_FollowWithLocalRejected(t *testing.T) {
	cmd := withMocks()
	err := cmd.Run(ReviewFlags{Follow: true, Local: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--follow flag is not applicable with --local flag")
}

func TestRun_CurrentBranchFailure(t *testing.T) {
	mockGit := &mockGitClient{
		currentBranchFunc: func() (string, error) { return "", errMock },
	}
	cmd := withMocks(withGit(mockGit))
	err := cmd.Run(ReviewFlags{Local: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current branch")
}

// ---------------------------------------------------------------------------
// Build commit message
// ---------------------------------------------------------------------------

func TestBuildCommitMessage_NoSummary(t *testing.T) {
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "summary.txt")

	cmd := withMocks()
	msg := cmd.buildCommitMessage(0, summaryPath)
	assert.Equal(t, "review: item-0", msg)
}

func TestBuildCommitMessage_WithSummary(t *testing.T) {
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "summary.txt")
	require.NoError(t, os.WriteFile(summaryPath, []byte("Fixed the bug"), 0644))

	cmd := withMocks()
	msg := cmd.buildCommitMessage(0, summaryPath)
	assert.Equal(t, "review: item-0 Fixed the bug", msg)
}
