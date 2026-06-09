package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orchestrationComment "github.com/zon/ralph/internal/orchestration/comment"
)

// ---------------------------------------------------------------------------
// workflowCommentAIClient tests
// ---------------------------------------------------------------------------

func TestWorkflowCommentAIClient_RenderCommentPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		ctx             orchestrationComment.CommentContext
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "includes all comment context fields",
			ctx: orchestrationComment.CommentContext{
				CommentBody: "please fix this bug",
				PRNumber:    42,
				PRBranch:    "fix-bug",
				RepoOwner:   "my-org",
				RepoName:    "my-repo",
			},
			wantContains: []string{
				"Comment: please fix this bug",
				"PR Number: 42",
				"PR Branch: fix-bug",
				"Repo: my-org/my-repo",
			},
		},
		{
			name: "includes default instructions when no file provided",
			ctx: orchestrationComment.CommentContext{
				CommentBody: "test",
				PRNumber:    1,
				PRBranch:    "main",
				RepoOwner:   "owner",
				RepoName:    "repo",
			},
			wantContains: []string{
				"Comment: test",
				"PR Number: 1",
				"PR Branch: main",
				"Repo: owner/repo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &workflowCommentAIClient{}
			prompt, err := client.RenderCommentPrompt(tt.ctx, "")
			require.NoError(t, err)
			for _, want := range tt.wantContains {
				assert.Contains(t, prompt, want)
			}
			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, prompt, notWant)
			}
		})
	}
}

func TestWorkflowCommentAIClient_RenderCommentPrompt_WithInstructionsFile(t *testing.T) {
	dir := t.TempDir()
	instructions := "custom: fix the issue"
	path := filepath.Join(dir, "instructions.md")
	require.NoError(t, os.WriteFile(path, []byte(instructions), 0644))

	client := &workflowCommentAIClient{}
	ctx := orchestrationComment.CommentContext{
		CommentBody: "test",
		PRNumber:    1,
		PRBranch:    "main",
		RepoOwner:   "owner",
		RepoName:    "repo",
	}
	prompt, err := client.RenderCommentPrompt(ctx, path)
	require.NoError(t, err)
	assert.Contains(t, prompt, "custom: fix the issue")
}

func TestWorkflowCommentAIClient_GenerateCommentReply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		ctx          orchestrationComment.CommentContext
		pushed       bool
		wantContains string
	}{
		{
			name: "pushed returns success message",
			ctx: orchestrationComment.CommentContext{
				PRNumber: 42,
			},
			pushed:       true,
			wantContains: "completed successfully for PR #42",
		},
		{
			name: "not pushed returns no changes message",
			ctx: orchestrationComment.CommentContext{
				PRNumber: 99,
			},
			pushed:       false,
			wantContains: "completed with no changes to push for PR #99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &workflowCommentAIClient{}
			reply, err := client.GenerateCommentReply(tt.ctx, tt.pushed)
			require.NoError(t, err)
			assert.Contains(t, reply, tt.wantContains)
		})
	}
}

// ---------------------------------------------------------------------------
// workflowCommentGitClient tests
// ---------------------------------------------------------------------------

func TestWorkflowCommentGitClient_ReportExists(t *testing.T) {
	t.Parallel()

	t.Run("returns false when report.md does not exist", func(t *testing.T) {
		dir := t.TempDir()
		origDir := chdir(t, dir)
		defer chdir(t, origDir)

		client := &workflowCommentGitClient{}
		assert.False(t, client.ReportExists())
	})

	t.Run("returns true when report.md exists", func(t *testing.T) {
		dir := t.TempDir()
		origDir := chdir(t, dir)
		defer chdir(t, origDir)

		require.NoError(t, os.WriteFile("report.md", []byte("test report"), 0644))

		client := &workflowCommentGitClient{}
		assert.True(t, client.ReportExists())
	})
}

func TestWorkflowCommentGitClient_HasChanges(t *testing.T) {
	dir := t.TempDir()
	origDir := chdir(t, dir)
	defer chdir(t, origDir)

	initGitRepo(t)
	createAndCommitFile(t, "initial.txt", "initial content")

	t.Run("returns false when no uncommitted changes", func(t *testing.T) {
		client := &workflowCommentGitClient{}
		assert.False(t, client.HasChanges())
	})

	t.Run("returns true when there are uncommitted changes", func(t *testing.T) {
		require.NoError(t, os.WriteFile("new-file.txt", []byte("new content"), 0644))
		client := &workflowCommentGitClient{}
		assert.True(t, client.HasChanges())
	})
}

func TestWorkflowCommentGitClient_CommitAndPushFromReport(t *testing.T) {
	dir := t.TempDir()
	origDir := chdir(t, dir)
	defer chdir(t, origDir)

	initGitRepo(t)
	createAndCommitFile(t, "initial.txt", "initial content")

	t.Run("fails when no changes to commit", func(t *testing.T) {
		client := &workflowCommentGitClient{}
		err := client.CommitAndPushFromReport()
		require.Error(t, err)
	})

	t.Run("fails with empty commit message when there are changes", func(t *testing.T) {
		require.NoError(t, os.WriteFile("report.md", []byte("test report"), 0644))
		require.NoError(t, os.WriteFile("other.md", []byte("other"), 0644))

		client := &workflowCommentGitClient{}
		err := client.CommitAndPushFromReport()
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func chdir(t *testing.T, dir string) string {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	return orig
}

func initGitRepo(t *testing.T) {
	t.Helper()
	runGitCmd(t, "init")
	runGitCmd(t, "config", "user.email", "test@test.com")
	runGitCmd(t, "config", "user.name", "Test")
}

func createAndCommitFile(t *testing.T, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(name, []byte(content), 0644))
	runGitCmd(t, "add", ".")
	runGitCmd(t, "commit", "-m", "add "+name)
}

func runGitCmd(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, strings.TrimSpace(string(out)))
	return strings.TrimSpace(string(out))
}
