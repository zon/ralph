package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig builds a minimal Config suitable for server tests.
func testConfig() *Config {
	return &Config{
		App: AppConfig{
			Port:          8080,
			RalphUsername: "ralph-bot",
			Repos: []RepoConfig{
				{Owner: "acme", Name: "myrepo", ClonePath: "/repos/myrepo"},
			},
		},
		Secrets: Secrets{
			GitHubToken: "ghp_test",
			Repos: []RepoSecret{
				{Owner: "acme", Name: "myrepo", WebhookSecret: "supersecret"},
			},
		},
	}
}

// sign returns a valid X-Hub-Signature-256 header value for body using secret.
func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// buildPayload creates a JSON payload with the given repository owner/name and
// merges any extra top-level fields provided.
func buildPayload(owner, name string, extra map[string]interface{}) []byte {
	payload := map[string]interface{}{
		"repository": map[string]interface{}{
			"name": name,
			"owner": map[string]interface{}{
				"login": owner,
			},
		},
	}
	for k, v := range extra {
		payload[k] = v
	}
	b, _ := json.Marshal(payload)
	return b
}

// postWebhook sends a POST /webhook request to the server and returns the recorder.
func postWebhook(t *testing.T, s *Server, eventType string, body []byte, signature string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", eventType)
	if signature != "" {
		req.Header.Set("X-Hub-Signature-256", signature)
	}
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)
	return w
}

// ──────────────────────────────────────────────────────────────────────────────
// Signature validation tests
// ──────────────────────────────────────────────────────────────────────────────

func TestHandleWebhook_MissingSignature_Returns401(t *testing.T) {
	s := NewServer(testConfig(), nil)
	body := buildPayload("acme", "myrepo", nil)
	w := postWebhook(t, s, "pull_request_review_comment", body, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_InvalidSignature_Returns401(t *testing.T) {
	s := NewServer(testConfig(), nil)
	body := buildPayload("acme", "myrepo", nil)
	w := postWebhook(t, s, "pull_request_review_comment", body, "sha256=deadbeef")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_WrongPrefixSignature_Returns401(t *testing.T) {
	s := NewServer(testConfig(), nil)
	body := buildPayload("acme", "myrepo", nil)
	// Use sha1= prefix instead of sha256=
	w := postWebhook(t, s, "pull_request_review_comment", body, "sha1=abc123")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_ValidSignature_Returns200(t *testing.T) {
	s := NewServer(testConfig(), nil)
	body := buildPayload("acme", "myrepo", nil)
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleWebhook_UnknownRepo_Returns401(t *testing.T) {
	s := NewServer(testConfig(), nil)
	body := buildPayload("unknown-org", "other-repo", nil)
	sig := sign(body, "doesnotmatter")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ──────────────────────────────────────────────────────────────────────────────
// Event type filtering tests
// ──────────────────────────────────────────────────────────────────────────────

func TestHandleWebhook_IssueComment_Ignored200(t *testing.T) {
	var called bool
	s := NewServer(testConfig(), func(_ string, _ map[string]interface{}) {
		called = true
	})
	body := buildPayload("acme", "myrepo", nil)
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "issue_comment", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called, "handler should not be called for issue_comment events")
}

func TestHandleWebhook_UnrecognisedEvent_Ignored200(t *testing.T) {
	var called bool
	s := NewServer(testConfig(), func(_ string, _ map[string]interface{}) {
		called = true
	})
	body := buildPayload("acme", "myrepo", nil)
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "push", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called, "handler should not be called for unrecognised events")
}

// ──────────────────────────────────────────────────────────────────────────────
// Event filtering: PR author and event poster checks
// ──────────────────────────────────────────────────────────────────────────────

func TestHandleWebhook_ReviewComment_PRNotOpenedByRalph_Ignored200(t *testing.T) {
	var called bool
	s := NewServer(testConfig(), func(_ string, _ map[string]interface{}) {
		called = true
	})
	// PR opened by a human, not ralph-bot
	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{"user": map[string]interface{}{"login": "human-developer"}},
		"comment":      map[string]interface{}{"user": map[string]interface{}{"login": "another-user"}},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called, "handler should not be called when PR was not opened by ralph")
}

func TestHandleWebhook_ReviewComment_PostedByRalph_Ignored200(t *testing.T) {
	var called bool
	s := NewServer(testConfig(), func(_ string, _ map[string]interface{}) {
		called = true
	})
	// PR opened by ralph-bot but comment also posted by ralph-bot
	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{"user": map[string]interface{}{"login": "ralph-bot"}},
		"comment":      map[string]interface{}{"user": map[string]interface{}{"login": "ralph-bot"}},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called, "handler should not be called when comment was posted by ralph")
}

func TestHandleWebhook_ReviewComment_RalphPRCaseInsensitive_HandlerCalled(t *testing.T) {
	var called bool
	s := NewServer(testConfig(), func(_ string, _ map[string]interface{}) {
		called = true
	})
	// PR opened by ralph-bot (different case) and comment from a human
	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{"user": map[string]interface{}{"login": "Ralph-Bot"}},
		"comment":      map[string]interface{}{"user": map[string]interface{}{"login": "human-reviewer"}},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "handler should be called when PR author matches ralph (case-insensitive)")
}

func TestHandleWebhook_PullRequestReview_PRNotOpenedByRalph_Ignored200(t *testing.T) {
	var called bool
	s := NewServer(testConfig(), func(_ string, _ map[string]interface{}) {
		called = true
	})
	// PR opened by human, not ralph-bot
	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{"user": map[string]interface{}{"login": "human-developer"}},
		"review":       map[string]interface{}{"state": "approved", "user": map[string]interface{}{"login": "another-reviewer"}},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called, "handler should not be called when PR was not opened by ralph")
}

func TestHandleWebhook_PullRequestReview_ReviewPostedByRalph_Ignored200(t *testing.T) {
	var called bool
	s := NewServer(testConfig(), func(_ string, _ map[string]interface{}) {
		called = true
	})
	// PR opened by ralph-bot but review also posted by ralph-bot
	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{"user": map[string]interface{}{"login": "ralph-bot"}},
		"review":       map[string]interface{}{"state": "approved", "user": map[string]interface{}{"login": "ralph-bot"}},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called, "handler should not be called when review was posted by ralph")
}

func TestHandleWebhook_PullRequestReviewComment_HandlerCalled(t *testing.T) {
	var receivedEvent string
	s := NewServer(testConfig(), func(eventType string, _ map[string]interface{}) {
		receivedEvent = eventType
	})
	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{"user": map[string]interface{}{"login": "ralph-bot"}},
		"comment":      map[string]interface{}{"user": map[string]interface{}{"login": "human-user"}},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pull_request_review_comment", receivedEvent)
}

// ──────────────────────────────────────────────────────────────────────────────
// pull_request_review state filtering
// ──────────────────────────────────────────────────────────────────────────────

func TestHandleWebhook_PullRequestReview_ApprovedState_HandlerCalled(t *testing.T) {
	var receivedEvent string
	s := NewServer(testConfig(), func(eventType string, _ map[string]interface{}) {
		receivedEvent = eventType
	})
	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{"user": map[string]interface{}{"login": "ralph-bot"}},
		"review":       map[string]interface{}{"state": "approved", "user": map[string]interface{}{"login": "human-user"}},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pull_request_review", receivedEvent)
}

func TestHandleWebhook_PullRequestReview_ApprovedStateCaseInsensitive_HandlerCalled(t *testing.T) {
	var receivedEvent string
	s := NewServer(testConfig(), func(eventType string, _ map[string]interface{}) {
		receivedEvent = eventType
	})
	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{"user": map[string]interface{}{"login": "ralph-bot"}},
		"review":       map[string]interface{}{"state": "APPROVED", "user": map[string]interface{}{"login": "human-user"}},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pull_request_review", receivedEvent)
}

func TestHandleWebhook_PullRequestReview_NonApprovedState_Ignored200(t *testing.T) {
	states := []string{"changes_requested", "commented", "dismissed", ""}
	for _, state := range states {
		t.Run(fmt.Sprintf("state=%q", state), func(t *testing.T) {
			var called bool
			s := NewServer(testConfig(), func(_ string, _ map[string]interface{}) {
				called = true
			})
			body := buildPayload("acme", "myrepo", map[string]interface{}{
				"review": map[string]interface{}{"state": state},
			})
			sig := sign(body, "supersecret")
			w := postWebhook(t, s, "pull_request_review", body, sig)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.False(t, called, "handler must not be called for state %q", state)
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Handler receives correct payload
// ──────────────────────────────────────────────────────────────────────────────

func TestHandleWebhook_HandlerReceivesPayload(t *testing.T) {
	var receivedPayload map[string]interface{}
	s := NewServer(testConfig(), func(_ string, payload map[string]interface{}) {
		receivedPayload = payload
	})
	body := buildPayload("acme", "myrepo", map[string]interface{}{
		"pull_request": map[string]interface{}{"user": map[string]interface{}{"login": "ralph-bot"}},
		"comment": map[string]interface{}{
			"body": "please fix the tests",
			"user": map[string]interface{}{"login": "human-user"},
		},
	})
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, receivedPayload)
	comment, ok := receivedPayload["comment"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "please fix the tests", comment["body"])
}

// ──────────────────────────────────────────────────────────────────────────────
// validateSignature unit tests
// ──────────────────────────────────────────────────────────────────────────────

func TestValidateSignature(t *testing.T) {
	secret := "mysecret"
	body := []byte(`{"test":"value"}`)
	validSig := sign(body, secret)

	tests := []struct {
		name      string
		signature string
		want      bool
	}{
		{"valid signature", validSig, true},
		{"missing signature", "", false},
		{"wrong prefix", "sha1=" + validSig[7:], false},
		{"tampered body", "sha256=000000", false},
		{"wrong secret", sign(body, "wrong"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateSignature(body, secret, tc.signature)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// extractRepoFromPayload unit tests
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractRepoFromPayload(t *testing.T) {
	tests := []struct {
		name      string
		body      []byte
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{
			name:      "valid payload",
			body:      buildPayload("octocat", "Hello-World", nil),
			wantOwner: "octocat",
			wantName:  "Hello-World",
		},
		{
			name:    "invalid JSON",
			body:    []byte("not json"),
			wantErr: true,
		},
		{
			name:      "missing repository",
			body:      []byte(`{}`),
			wantOwner: "",
			wantName:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			owner, name, err := extractRepoFromPayload(tc.body)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantOwner, owner)
			assert.Equal(t, tc.wantName, name)
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// nestedString unit tests
// ──────────────────────────────────────────────────────────────────────────────

func TestNestedString(t *testing.T) {
	m := map[string]interface{}{
		"review": map[string]interface{}{
			"state": "approved",
		},
	}

	tests := []struct {
		name   string
		keys   []string
		want   string
		wantOk bool
	}{
		{"existing key chain", []string{"review", "state"}, "approved", true},
		{"missing top key", []string{"missing"}, "", false},
		{"missing nested key", []string{"review", "nope"}, "", false},
		{"value not string", []string{"review"}, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := nestedString(m, tc.keys...)
			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// NewServer / port configuration
// ──────────────────────────────────────────────────────────────────────────────

func TestNewServer_UsesConfiguredPort(t *testing.T) {
	cfg := testConfig()
	cfg.App.Port = 9090
	s := NewServer(cfg, nil)
	assert.NotNil(t, s)
	assert.Equal(t, 9090, s.config.App.Port)
}
