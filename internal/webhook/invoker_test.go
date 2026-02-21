package webhook

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────────────────────────────────
// buildInstructions tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildInstructions_ContainsCommentBody(t *testing.T) {
	comment := "please fix the failing test in foo_test.go"
	result := buildInstructions(comment)
	assert.Contains(t, result, comment, "instructions should include the original comment text")
}

func TestBuildInstructions_ContainsRequiredDirectives(t *testing.T) {
	result := buildInstructions("do something")
	assert.Contains(t, result, "commit and push", "instructions must direct agent to commit and push")
	assert.Contains(t, result, "GitHub PR comment", "instructions must direct agent to post a PR comment")
}

func TestBuildInstructions_ContainsAnswerDirective(t *testing.T) {
	result := buildInstructions("What does this function do?")
	assert.Contains(t, result, "answer", "instructions should direct the agent to answer questions")
}

// ──────────────────────────────────────────────────────────────────────────────
// projectFileFromBranch tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProjectFileFromBranch(t *testing.T) {
	tests := []struct {
		name      string
		clonePath string
		branch    string
		want      string
	}{
		{
			name:      "ralph-prefixed branch",
			clonePath: "/repos/myrepo",
			branch:    "ralph/my-feature",
			want:      "/repos/myrepo/projects/my-feature.yaml",
		},
		{
			name:      "ralph-prefixed branch with dashes",
			clonePath: "/repos/myrepo",
			branch:    "ralph/github-webhook-service",
			want:      "/repos/myrepo/projects/github-webhook-service.yaml",
		},
		{
			name:      "non-ralph branch falls back to full name",
			clonePath: "/repos/myrepo",
			branch:    "feature/something",
			want:      "/repos/myrepo/projects/feature-something.yaml",
		},
		{
			name:      "empty branch",
			clonePath: "/repos/myrepo",
			branch:    "",
			want:      "/repos/myrepo/projects/.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := projectFileFromBranch(tc.clonePath, tc.branch)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Invoker dry-run tests
// ──────────────────────────────────────────────────────────────────────────────

func TestInvoker_InvokeRalphRun_DryRun(t *testing.T) {
	inv := NewInvoker(true)
	err := inv.InvokeRalphRun("/repos/myrepo/projects/my-feature.yaml", "please fix the tests")
	require.NoError(t, err)

	require.NotNil(t, inv.LastInvoke)
	assert.Equal(t, "run", inv.LastInvoke.Command)
	assert.Contains(t, inv.LastInvoke.InstructionsContent, "please fix the tests",
		"instructions content should include the comment body")
	assert.Equal(t, "/repos/myrepo/projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Contains(t, inv.LastInvoke.Args, "--remote")
	assert.Contains(t, inv.LastInvoke.Args, "--no-notify")
	assert.Contains(t, inv.LastInvoke.Args, "--instructions")
}

func TestInvoker_InvokeRalphMerge_DryRun(t *testing.T) {
	inv := NewInvoker(true)
	err := inv.InvokeRalphMerge("/repos/myrepo/projects/my-feature.yaml", "ralph/my-feature")
	require.NoError(t, err)

	require.NotNil(t, inv.LastInvoke)
	assert.Equal(t, "merge", inv.LastInvoke.Command)
	assert.Equal(t, "/repos/myrepo/projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Equal(t, "ralph/my-feature", inv.LastInvoke.Args[1])
	assert.Empty(t, inv.LastInvoke.InstructionsContent)
}

// ──────────────────────────────────────────────────────────────────────────────
// HandleEvent tests (end-to-end with dry-run invoker + server)
// ──────────────────────────────────────────────────────────────────────────────

// testConfigWithClonePath returns a Config that includes a ClonePath for the
// acme/myrepo repository, suitable for HandleEvent tests.
func testConfigWithClonePath() *Config {
	return &Config{
		App: AppConfig{
			Port:          8080,
			RalphUsername: "ralph-bot",
			Repos: []RepoConfig{
				{Owner: "acme", Name: "myrepo", ClonePath: "/repos/myrepo"},
			},
		},
		Secrets: Secrets{
			Repos: []RepoSecret{
				{Owner: "acme", Name: "myrepo", WebhookSecret: "supersecret"},
			},
		},
	}
}

func TestHandleEvent_CommentEvent_InvokesRalphRun(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfigWithClonePath()

	handler := inv.HandleEvent(cfg)
	payload := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-feature"},
			"user": map[string]interface{}{"login": "ralph-bot"},
		},
		"comment": map[string]interface{}{
			"body": "please add a unit test",
			"user": map[string]interface{}{"login": "human-reviewer"},
		},
	}

	handler("pull_request_review_comment", "acme", "myrepo", payload)

	require.NotNil(t, inv.LastInvoke, "invoker should have been called")
	assert.Equal(t, "run", inv.LastInvoke.Command)
	assert.Equal(t, "/repos/myrepo/projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Contains(t, inv.LastInvoke.InstructionsContent, "please add a unit test")
}

func TestHandleEvent_ApprovalEvent_InvokesRalphMerge(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfigWithClonePath()

	handler := inv.HandleEvent(cfg)
	payload := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-feature"},
			"user": map[string]interface{}{"login": "ralph-bot"},
		},
		"review": map[string]interface{}{
			"state": "approved",
			"user":  map[string]interface{}{"login": "human-reviewer"},
		},
	}

	handler("pull_request_review", "acme", "myrepo", payload)

	require.NotNil(t, inv.LastInvoke, "invoker should have been called")
	assert.Equal(t, "merge", inv.LastInvoke.Command)
	assert.Equal(t, "/repos/myrepo/projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Equal(t, "ralph/my-feature", inv.LastInvoke.Args[1])
}

func TestHandleEvent_UnknownRepo_NoInvocation(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfigWithClonePath()

	handler := inv.HandleEvent(cfg)
	payload := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-feature"},
		},
	}

	// Unknown owner/repo – should be a no-op.
	handler("pull_request_review_comment", "unknown-org", "unknown-repo", payload)

	assert.Nil(t, inv.LastInvoke, "invoker must not be called for unknown repos")
}

func TestHandleEvent_CommentEvent_InstructionsIncludeDirectives(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfigWithClonePath()
	handler := inv.HandleEvent(cfg)

	commentText := "Can you explain what this function does?"
	payload := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/some-project"},
		},
		"comment": map[string]interface{}{
			"body": commentText,
		},
	}

	handler("pull_request_review_comment", "acme", "myrepo", payload)

	require.NotNil(t, inv.LastInvoke)
	instructions := inv.LastInvoke.InstructionsContent
	assert.Contains(t, instructions, commentText)
	// Must direct the agent to post a summary PR comment
	assert.True(t,
		strings.Contains(instructions, "PR comment") || strings.Contains(instructions, "pr comment"),
		"instructions should mention posting a PR comment")
}

// ──────────────────────────────────────────────────────────────────────────────
// Integration: full server → invoker path (dry-run)
// ──────────────────────────────────────────────────────────────────────────────

func TestServer_CommentEvent_TriggersRalphRun(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfigWithClonePath()
	s := NewServer(cfg, inv.HandleEvent(cfg))

	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{
			"user": map[string]interface{}{"login": "ralph-bot"},
			"head": map[string]interface{}{"ref": "ralph/my-project"},
		},
		"comment": map[string]interface{}{
			"body": "please refactor this",
			"user": map[string]interface{}{"login": "human-dev"},
		},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)

	assert.Equal(t, 200, w.Code)
	require.NotNil(t, inv.LastInvoke)
	assert.Equal(t, "run", inv.LastInvoke.Command)
	assert.Contains(t, inv.LastInvoke.InstructionsContent, "please refactor this")
}

func TestServer_ApprovalEvent_TriggersRalphMerge(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfigWithClonePath()
	s := NewServer(cfg, inv.HandleEvent(cfg))

	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{
			"user": map[string]interface{}{"login": "ralph-bot"},
			"head": map[string]interface{}{"ref": "ralph/my-project"},
		},
		"review": map[string]interface{}{
			"state": "approved",
			"user":  map[string]interface{}{"login": "human-reviewer"},
		},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review", body, sig)

	assert.Equal(t, 200, w.Code)
	require.NotNil(t, inv.LastInvoke)
	assert.Equal(t, "merge", inv.LastInvoke.Command)
	assert.Equal(t, "ralph/my-project", inv.LastInvoke.Args[1])
}
