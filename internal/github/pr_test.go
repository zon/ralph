package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
)

// Compile-time assertion that *GH implements GHClient.
var _ GHClient = (*GH)(nil)

// mockGH is a minimal GHClient stub for unit tests.
type mockGH struct{}

func (m *mockGH) IsReady() bool                                        { return true }
func (m *mockGH) FindExistingPR(head string) (string, error)           { return "", nil }
func (m *mockGH) CreatePR(title, body, base, head string) (string, error) {
	return "https://github.com/mock/repo/pull/1", nil
}
func (m *mockGH) GetPRHeadRefOid(pr string) (string, error)           { return "abc123", nil }
func (m *mockGH) MergePR(pr, repo string) error                       { return nil }

func TestCreatePullRequest_UsesTitleAsPRTitle(t *testing.T) {
	proj := &project.Project{
		Slug:  "test-project",
		Title: "This is a detailed title",
	}
	ctx := context.NewContext()

	prURL, err := CreatePullRequest(&mockGH{}, ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
	assert.Contains(t, prURL, "github.com")
}

func TestCreatePullRequest_UsesSlugWhenTitleEmpty(t *testing.T) {
	proj := &project.Project{
		Slug:  "my-project",
		Title: "",
	}
	ctx := context.NewContext()

	prURL, err := CreatePullRequest(&mockGH{}, ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
}

func TestCreatePullRequest_UsesSlugWhenTitleMissing(t *testing.T) {
	proj := &project.Project{
		Slug: "fallback-project",
	}
	ctx := context.NewContext()

	prURL, err := CreatePullRequest(&mockGH{}, ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.NotEmpty(t, prURL)
}

func TestCreatePullRequest_DelegatesToIsReady(t *testing.T) {
	called := false
	m := &mockGHWithCheck{readyCheck: func() bool {
		called = true
		return true
	}}
	proj := &project.Project{Slug: "test", Title: "Test"}
	ctx := context.NewContext()

	_, err := CreatePullRequest(m, ctx, proj, "feature-branch", "main", "PR body")
	assert.NoError(t, err)
	assert.True(t, called, "expected GHClient.IsReady to be called")
}

type mockGHWithCheck struct {
	readyCheck func() bool
}

func (m *mockGHWithCheck) IsReady() bool                                        { return m.readyCheck() }
func (m *mockGHWithCheck) FindExistingPR(head string) (string, error)           { return "", nil }
func (m *mockGHWithCheck) CreatePR(title, body, base, head string) (string, error) {
	return "https://github.com/mock/repo/pull/1", nil
}
func (m *mockGHWithCheck) GetPRHeadRefOid(pr string) (string, error)           { return "abc123", nil }
func (m *mockGHWithCheck) MergePR(pr, repo string) error                       { return nil }
