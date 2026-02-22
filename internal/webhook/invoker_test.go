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
		name   string
		branch string
		want   string
	}{
		{
			name:   "ralph-prefixed branch",
			branch: "ralph/my-feature",
			want:   "projects/my-feature.yaml",
		},
		{
			name:   "ralph-prefixed branch with dashes",
			branch: "ralph/github-webhook-service",
			want:   "projects/github-webhook-service.yaml",
		},
		{
			name:   "non-ralph branch falls back to full name",
			branch: "feature/something",
			want:   "projects/feature-something.yaml",
		},
		{
			name:   "empty branch",
			branch: "",
			want:   "projects/.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := projectFileFromBranch(tc.branch)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Invoker dry-run tests
// ──────────────────────────────────────────────────────────────────────────────

func TestInvoker_InvokeRalphRun_DryRun(t *testing.T) {
	inv := NewInvoker(true)
	err := inv.InvokeRalphRun("projects/my-feature.yaml", "please fix the tests")
	require.NoError(t, err)

	require.NotNil(t, inv.LastInvoke)
	assert.Equal(t, "run", inv.LastInvoke.Command)
	assert.Contains(t, inv.LastInvoke.InstructionsContent, "please fix the tests",
		"instructions content should include the comment body")
	assert.Equal(t, "projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Contains(t, inv.LastInvoke.Args, "--remote")
	assert.Contains(t, inv.LastInvoke.Args, "--no-notify")
	assert.Contains(t, inv.LastInvoke.Args, "--instructions")
}

func TestInvoker_InvokeRalphMerge_DryRun(t *testing.T) {
	inv := NewInvoker(true)
	err := inv.InvokeRalphMerge("projects/my-feature.yaml", "ralph/my-feature")
	require.NoError(t, err)

	require.NotNil(t, inv.LastInvoke)
	assert.Equal(t, "merge", inv.LastInvoke.Command)
	assert.Equal(t, "projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Equal(t, "ralph/my-feature", inv.LastInvoke.Args[1])
	assert.Empty(t, inv.LastInvoke.InstructionsContent)
}

// ──────────────────────────────────────────────────────────────────────────────
// HandleEvent tests (end-to-end with dry-run invoker + server)
// ──────────────────────────────────────────────────────────────────────────────

func TestHandleEvent_CommentEvent_InvokesRalphRun(t *testing.T) {
	inv := NewInvoker(true)
	handler := inv.HandleEvent()
	payload := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-feature"},
		},
		"comment": map[string]interface{}{
			"body": "please add a unit test",
			"user": map[string]interface{}{"login": "human-reviewer"},
		},
	}

	handler("pull_request_review_comment", "acme", "myrepo", payload)

	require.NotNil(t, inv.LastInvoke, "invoker should have been called")
	assert.Equal(t, "run", inv.LastInvoke.Command)
	assert.Equal(t, "projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Contains(t, inv.LastInvoke.InstructionsContent, "please add a unit test")
}

func TestHandleEvent_IssueCommentEvent_InvokesRalphRun(t *testing.T) {
	inv := NewInvoker(true)
	handler := inv.HandleEvent()
	// issue_comment payloads use issue.pull_request.url for PR detection and
	// pull_request.head.ref for the branch (when included by the sender).
	payload := map[string]interface{}{
		"issue": map[string]interface{}{
			"number": 42,
			"pull_request": map[string]interface{}{
				"url": "https://api.github.com/repos/acme/myrepo/pulls/42",
			},
		},
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-feature"},
		},
		"comment": map[string]interface{}{
			"body": "please fix the tests",
			"user": map[string]interface{}{"login": "human-user"},
		},
	}

	handler("issue_comment", "acme", "myrepo", payload)

	require.NotNil(t, inv.LastInvoke, "invoker should have been called for issue_comment")
	assert.Equal(t, "run", inv.LastInvoke.Command)
	assert.Equal(t, "projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Contains(t, inv.LastInvoke.InstructionsContent, "please fix the tests")
}

func TestHandleEvent_ApprovalEvent_InvokesRalphMerge(t *testing.T) {
	inv := NewInvoker(true)
	handler := inv.HandleEvent()
	payload := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-feature"},
		},
		"review": map[string]interface{}{
			"state": "approved",
			"user":  map[string]interface{}{"login": "human-reviewer"},
		},
	}

	handler("pull_request_review", "acme", "myrepo", payload)

	require.NotNil(t, inv.LastInvoke, "invoker should have been called")
	assert.Equal(t, "merge", inv.LastInvoke.Command)
	assert.Equal(t, "projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Equal(t, "ralph/my-feature", inv.LastInvoke.Args[1])
}

func TestHandleEvent_ReviewCommentEvent_InvokesRalphRun(t *testing.T) {
	inv := NewInvoker(true)
	handler := inv.HandleEvent()
	payload := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-feature"},
		},
		"review": map[string]interface{}{
			"state": "commented",
			"body":  "please add error handling here",
			"user":  map[string]interface{}{"login": "human-reviewer"},
		},
	}

	handler("pull_request_review", "acme", "myrepo", payload)

	require.NotNil(t, inv.LastInvoke, "invoker should have been called for commented review")
	assert.Equal(t, "run", inv.LastInvoke.Command)
	assert.Equal(t, "projects/my-feature.yaml", inv.LastInvoke.Args[0])
	assert.Contains(t, inv.LastInvoke.InstructionsContent, "please add error handling here")
}

func TestHandleEvent_CommentEvent_InstructionsIncludeDirectives(t *testing.T) {
	inv := NewInvoker(true)
	handler := inv.HandleEvent()

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
	cfg := testConfig()
	s := NewServer(cfg, inv.HandleEvent())

	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{
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

func TestServer_IssueCommentEvent_TriggersRalphRun(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfig()
	s := NewServer(cfg, inv.HandleEvent())

	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"issue": map[string]interface{}{
			"number": 42,
			"pull_request": map[string]interface{}{
				"url": "https://api.github.com/repos/acme/myrepo/pulls/42",
			},
		},
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-project"},
		},
		"comment": map[string]interface{}{
			"body": "please update the docs",
			"user": map[string]interface{}{"login": "human-dev"},
		},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "issue_comment", body, sig)

	assert.Equal(t, 200, w.Code)
	require.NotNil(t, inv.LastInvoke)
	assert.Equal(t, "run", inv.LastInvoke.Command)
	assert.Contains(t, inv.LastInvoke.InstructionsContent, "please update the docs")
}

func TestServer_ReviewCommentEvent_TriggersRalphRun(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfig()
	s := NewServer(cfg, inv.HandleEvent())

	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-project"},
		},
		"review": map[string]interface{}{
			"state": "commented",
			"body":  "please simplify this function",
			"user":  map[string]interface{}{"login": "human-reviewer"},
		},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review", body, sig)

	assert.Equal(t, 200, w.Code)
	require.NotNil(t, inv.LastInvoke)
	assert.Equal(t, "run", inv.LastInvoke.Command)
	assert.Contains(t, inv.LastInvoke.InstructionsContent, "please simplify this function")
}

func TestServer_ReviewCommentEvent_EmptyBody_NoInvoke(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfig()
	s := NewServer(cfg, inv.HandleEvent())

	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{"ref": "ralph/my-project"},
		},
		"review": map[string]interface{}{
			"state": "commented",
			"body":  "",
			"user":  map[string]interface{}{"login": "human-reviewer"},
		},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review", body, sig)

	assert.Equal(t, 200, w.Code)
	assert.Nil(t, inv.LastInvoke, "invoker should not be called for empty review body")
}

func TestServer_ApprovalEvent_TriggersRalphMerge(t *testing.T) {
	inv := NewInvoker(true)
	cfg := testConfig()
	s := NewServer(cfg, inv.HandleEvent())

	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{
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
