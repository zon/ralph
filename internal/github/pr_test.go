package github

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

// Compile-time assertion that *GH implements GHClient.
var _ GHClient = (*GH)(nil)

var testOut = output.NewClient(os.Stdout, os.Stderr, false)

func TestSplitTitle(t *testing.T) {
	tests := []struct {
		name      string
		summary   string
		wantTitle string
		wantBody  string
	}{
		{
			name:      "title and body",
			summary:   "# Add CSV export\n\nAdds the endpoint.\n",
			wantTitle: "Add CSV export",
			wantBody:  "Adds the endpoint.\n",
		},
		{
			name:      "title only",
			summary:   "# Add CSV export\n",
			wantTitle: "Add CSV export",
			wantBody:  "",
		},
		{
			name:      "body without title",
			summary:   "Adds the endpoint.\n",
			wantTitle: "",
			wantBody:  "Adds the endpoint.\n",
		},
		{
			name:      "empty heading",
			summary:   "# \nBody\n",
			wantTitle: "",
			wantBody:  "# \nBody\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, body := splitTitle(tt.summary)
			assert.Equal(t, tt.wantTitle, title)
			assert.Equal(t, tt.wantBody, body)
		})
	}
}

func TestCreatePullRequest_UsesSummaryTitleAsPRTitle(t *testing.T) {
	var capturedTitle, capturedBody string
	mock := &MockGH{
		IsReadyFn: func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) {
			capturedTitle = title
			capturedBody = body
			return "https://github.com/mock/repo/pull/1", nil
		},
	}
	NewClient(context.NewContext(), "main", mock, &opencode.MockOC{})

	proj := &project.Project{
		Slug:  "test-project",
		Title: "This is a detailed title",
	}
	summary := "# Add CSV export\n\nAdds an export endpoint.\n\n## Changes\n\n- Add serializer\n- Add route\n"
	prURL, err := CreatePullRequest(testOut, mock, proj, "feature-branch", "main", summary)
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
	assert.Contains(t, prURL, "github.com")
	assert.Equal(t, "Add CSV export", capturedTitle)
	assert.Equal(t, "Adds an export endpoint.\n\n## Changes\n\n- Add serializer\n- Add route\n", capturedBody)
}

func TestCreatePullRequest_FallsBackToProjectTitleWhenSummaryHasNoTitle(t *testing.T) {
	mock := &MockGH{
		IsReadyFn: func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) {
			assert.Equal(t, "This is a detailed title", title)
			assert.Equal(t, "PR body", body)
			return "https://github.com/mock/repo/pull/1", nil
		},
	}
	NewClient(context.NewContext(), "main", mock, &opencode.MockOC{})

	proj := &project.Project{
		Slug:  "test-project",
		Title: "This is a detailed title",
	}
	prURL, err := CreatePullRequest(testOut, mock, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
	assert.Contains(t, prURL, "github.com")
}

func TestCreatePullRequest_UsesSlugWhenTitleEmpty(t *testing.T) {
	mock := &MockGH{
		IsReadyFn: func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) {
			assert.Equal(t, "my-project", title)
			return "https://github.com/mock/repo/pull/1", nil
		},
	}
	NewClient(context.NewContext(), "main", mock, &opencode.MockOC{})

	proj := &project.Project{
		Slug:  "my-project",
		Title: "",
	}
	prURL, err := CreatePullRequest(testOut, mock, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
}

func TestCreatePullRequest_UsesSlugWhenTitleMissing(t *testing.T) {
	mock := &MockGH{
		IsReadyFn: func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) {
			assert.Equal(t, "fallback-project", title)
			return "https://github.com/mock/repo/pull/1", nil
		},
	}
	NewClient(context.NewContext(), "main", mock, &opencode.MockOC{})

	proj := &project.Project{
		Slug: "fallback-project",
	}
	prURL, err := CreatePullRequest(testOut, mock, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
}

func TestCreatePullRequest_DelegatesToIsReady(t *testing.T) {
	called := false
	mock := &MockGH{
		IsReadyFn: func() bool {
			called = true
			return true
		},
		CreatePRFn: func(title, body, base, head string) (string, error) {
			return "https://github.com/mock/repo/pull/1", nil
		},
	}
	NewClient(context.NewContext(), "main", mock, &opencode.MockOC{})

	proj := &project.Project{Slug: "test", Title: "Test"}
	_, err := CreatePullRequest(testOut, mock, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.True(t, called, "expected GHClient.IsReady to be called")
}
