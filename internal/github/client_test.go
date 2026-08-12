package github

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/opencode"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/project"
)

// mockCommitLog implements CommitLogGetter with a configurable commit log.
type mockCommitLog struct {
	getCommitLogFn func(base string, limit int) (string, error)
}

func (m *mockCommitLog) GetCommitLog(base string, limit int) (string, error) {
	if m.getCommitLogFn != nil {
		return m.getCommitLogFn(base, limit)
	}
	return "abc: feat\ndef: add\n", nil
}

// withMockCommitLog wires the mock commit log into the client.
func withMockCommitLog(client *Client, commitLog string) *Client {
	client.commitLog = &mockCommitLog{getCommitLogFn: func(string, int) (string, error) {
		return commitLog, nil
	}}
	return client
}

type mockGitAuthConfigurer struct {
	configureGitAuthFn func(ctx context.Context, owner, repo, secretsDir string) error
}

func (m *mockGitAuthConfigurer) ConfigureGitAuth(ctx context.Context, owner, repo, secretsDir string) error {
	if m.configureGitAuthFn != nil {
		return m.configureGitAuthFn(ctx, owner, repo, secretsDir)
	}
	return nil
}

func TestGitHubClientNew(t *testing.T) {
	ctx := execcontext.NewContext()
	client := NewClient(ctx, "main", NewGH(nil), &opencode.MockOC{})
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
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	mockOC := &opencode.MockOC{
		RunCommandFunc: func(_ context.Context, _, _, prompt string, _, _ io.Writer) error {
			if idx := strings.Index(prompt, "Write your summary to the file:"); idx >= 0 {
				rest := prompt[idx+len("Write your summary to the file:"):]
				rest = strings.TrimSpace(rest)
				lines := strings.SplitN(rest, "\n", 2)
				filePath := strings.TrimSpace(lines[0])
				os.WriteFile(filePath, []byte("Mock PR summary"), 0644)
			}
			return nil
		},
		GetStatsFunc: func() (opencode.Stats, error) {
			return opencode.Stats{}, nil
		},
	}
	client := withMockCommitLog(NewClient(ctx, "main", mock, mockOC), "abc: feat\n")
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	assert.NoError(t, err)
	assert.True(t, createPRCalled, "expected GHClient.CreatePR to be called")
}

func TestClientCreatePR_UsesTitleAsPRTitle(t *testing.T) {
	mock := &MockGH{
		IsReadyFn: func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) {
			assert.Equal(t, "This is a detailed title", title)
			return "https://github.com/mock/repo/pull/1", nil
		},
	}
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	mockOC := &opencode.MockOC{
		RunCommandFunc: func(_ context.Context, _, _, prompt string, _, _ io.Writer) error {
			return writeMockSummary(t, prompt)
		},
		GetStatsFunc: func() (opencode.Stats, error) { return opencode.Stats{}, nil },
	}
	client := withMockCommitLog(NewClient(ctx, "main", mock, mockOC), "abc: feat\n")
	proj := &project.Project{Slug: "some-branch", Title: "This is a detailed title"}

	err := client.CreatePR(proj)
	assert.NoError(t, err)
}

func TestClientCreatePR_SummaryGeneratedFromCommitLog(t *testing.T) {
	var capturedPrompt string
	mock := &MockGH{
		IsReadyFn:  func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) { return "https://github.com/o/r/p/1", nil },
	}
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	mockOC := &opencode.MockOC{
		RunCommandFunc: func(_ context.Context, _, _, prompt string, _, _ io.Writer) error {
			capturedPrompt = prompt
			return writeMockSummary(t, prompt)
		},
		GetStatsFunc: func() (opencode.Stats, error) { return opencode.Stats{}, nil },
	}
	client := withMockCommitLog(NewClient(ctx, "main", mock, mockOC), "abc: feat serializer\ndef: add endpoint\n")
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	assert.NoError(t, err)
	assert.Contains(t, capturedPrompt, "abc: feat serializer")
	assert.Contains(t, capturedPrompt, "def: add endpoint")
}

func TestClientCreatePR_SkipsWhenNoCommitsAhead(t *testing.T) {
	createPRCalled := false
	mock := &MockGH{
		IsReadyFn: func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) {
			createPRCalled = true
			return "https://github.com/o/r/p/1", nil
		},
	}
	var buf bytes.Buffer
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(&buf, &buf, true))
	mockOC := &opencode.MockOC{
		RunCommandFunc: func(_ context.Context, _, _, prompt string, _, _ io.Writer) error {
			return writeMockSummary(t, prompt)
		},
		GetStatsFunc: func() (opencode.Stats, error) { return opencode.Stats{}, nil },
	}
	client := withMockCommitLog(NewClient(ctx, "main", mock, mockOC), "")
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	assert.NoError(t, err)
	assert.False(t, createPRCalled, "expected PR creation to be skipped when no commits exist ahead of base")
	assert.Contains(t, buf.String(), "No commits ahead of base branch; skipping PR creation")
}

func TestClientCreatePR_SentinelNoCommitsSkipsWithoutError(t *testing.T) {
	mock := &MockGH{
		IsReadyFn:  func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) { return "", ErrNoCommitsBetweenBranches },
	}
	var buf bytes.Buffer
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(&buf, &buf, true))
	mockOC := &opencode.MockOC{
		RunCommandFunc: func(_ context.Context, _, _, prompt string, _, _ io.Writer) error {
			return writeMockSummary(t, prompt)
		},
		GetStatsFunc: func() (opencode.Stats, error) { return opencode.Stats{}, nil },
	}
	client := withMockCommitLog(NewClient(ctx, "main", mock, mockOC), "abc: feat\n")
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No commits ahead of base branch; skipping PR creation")
}

func TestClientCreatePR_WorkflowExecutionCallsConfigureGitAuth(t *testing.T) {
	called := false
	mockGitAuth := &mockGitAuthConfigurer{
		configureGitAuthFn: func(_ context.Context, owner, repo, _ string) error {
			called = true
			assert.Equal(t, "test-owner", owner)
			assert.Equal(t, "test-repo", repo)
			return errors.New("mock git auth error")
		},
	}
	mock := &MockGH{
		IsReadyFn:  func() bool { return true },
		CreatePRFn: func(title, body, base, head string) (string, error) { return "https://github.com/o/r/p/1", nil },
	}
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	ctx.SetWorkflowExecution(true)
	ctx.SetRepoOwner("test-owner")
	ctx.SetRepoName("test-repo")
	mockOC := &opencode.MockOC{
		RunCommandFunc: func(_ context.Context, _, _, prompt string, _, _ io.Writer) error {
			return writeMockSummary(t, prompt)
		},
		GetStatsFunc: func() (opencode.Stats, error) { return opencode.Stats{}, nil },
	}
	client := withMockCommitLog(NewClient(ctx, "main", mock, mockOC), "abc: feat\n")
	client.gitAuthConfigurer = mockGitAuth
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to refresh GitHub credentials")
	assert.True(t, called, "expected ConfigureGitAuth to be called")
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
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	mockOC := &opencode.MockOC{
		RunCommandFunc: func(_ context.Context, _, _, prompt string, _, _ io.Writer) error {
			return writeMockSummary(t, prompt)
		},
		GetStatsFunc: func() (opencode.Stats, error) { return opencode.Stats{}, nil },
	}
	client := withMockCommitLog(NewClient(ctx, "main", mock, mockOC), "abc: feat\n")
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
	ctx := execcontext.NewContext()
	ctx.SetOutput(output.NewClient(os.Stdout, os.Stderr, false))
	mockOC := &opencode.MockOC{
		RunCommandFunc: func(_ context.Context, _, _, prompt string, _, _ io.Writer) error {
			return writeMockSummary(t, prompt)
		},
		GetStatsFunc: func() (opencode.Stats, error) { return opencode.Stats{}, nil },
	}
	client := withMockCommitLog(NewClient(ctx, "main", mock, mockOC), "abc: feat\n")
	proj := &project.Project{Slug: "some-branch", Title: "Test Title"}

	err := client.CreatePR(proj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create pull request")
}

func writeMockSummary(t *testing.T, prompt string) error {
	t.Helper()
	if idx := strings.Index(prompt, "Write your summary to the file:"); idx >= 0 {
		rest := prompt[idx+len("Write your summary to the file:"):]
		rest = strings.TrimSpace(rest)
		lines := strings.SplitN(rest, "\n", 2)
		filePath := strings.TrimSpace(lines[0])
		os.WriteFile(filePath, []byte("Mock PR summary"), 0644)
	}
	return nil
}
