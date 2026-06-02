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

func TestCreatePullRequest_UsesTitleAsPRTitle(t *testing.T) {
	mock := &MockGH{
		IsReadyFn: func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) {
			assert.Equal(t, "This is a detailed title", title)
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
