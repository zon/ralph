package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/project"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
)

func TestGitHubClientNew(t *testing.T) {
	ctx := context.NewContext()
	client := NewClient(ctx, "main", &GH{})
	require.NotNil(t, client)
	var _ orchestrationRun.GitHubClient = client
}

func TestClientCreatePR_DelegatesToCreatePullRequest(t *testing.T) {
	createPRCalled := false
	mock := &MockGH{
		IsReadyFn: func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) {
			createPRCalled = true
			assert.Equal(t, "Test Title", title)
			assert.Equal(t, "main", base)
			assert.Equal(t, "some-branch", head)
			return "https://github.com/owner/repo/pull/1", nil
		},
	}
	ctx := context.NewContext()
	client := NewClient(ctx, "main", mock)
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	assert.NoError(t, err)
	assert.True(t, createPRCalled, "expected GHClient.CreatePR to be called")
}

func TestClientCreatePR_WorkflowExecutionCallsConfigureGitAuth(t *testing.T) {
	mock := &MockGH{
		IsReadyFn:  func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) { return "https://github.com/o/r/p/1", nil },
	}
	ctx := context.NewContext()
	ctx.SetWorkflowExecution(true)
	ctx.SetRepoOwner("test-owner")
	ctx.SetRepoName("test-repo")
	client := NewClient(ctx, "main", mock)
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	// ConfigureGitAuth will fail in test environment (no secrets), but we verify
	// the error is from the auth step, proving it was called.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to refresh GitHub credentials")
}

func TestClientCreatePR_SkipsConfigureGitAuthWhenNotWorkflow(t *testing.T) {
	createPRCalled := false
	mock := &MockGH{
		IsReadyFn: func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) {
			createPRCalled = true
			return "https://github.com/o/r/p/1", nil
		},
	}
	ctx := context.NewContext()
	client := NewClient(ctx, "main", mock)
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	assert.NoError(t, err)
	assert.True(t, createPRCalled, "expected CreatePR to succeed without ConfigureGitAuth")
}

func TestClientCreatePR_PropagatesCreatePullRequestError(t *testing.T) {
	mock := &MockGH{
		IsReadyFn:  func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) { return "", assert.AnError },
	}
	ctx := context.NewContext()
	client := NewClient(ctx, "main", mock)
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create pull request")
}
