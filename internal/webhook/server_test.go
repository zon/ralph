package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testConfig builds a minimal Config suitable for server tests.
func testConfig() *Config {
	return &Config{
		App: AppConfig{
			Port: 8080,
			Repos: []RepoConfig{
				{Owner: "acme", Name: "myrepo"},
			},
		},
		Secrets: Secrets{
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
// HTTP layer tests
// ──────────────────────────────────────────────────────────────────────────────

func TestHandleWebhook_InvalidJSON_Returns400(t *testing.T) {
	s := NewServer(testConfig())
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte("not json")))
	req.Header.Set("X-Hub-Signature-256", "sha256=anything")
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebhook_UnknownRepo_Returns401(t *testing.T) {
	s := NewServer(testConfig())
	body := buildPayload("unknown-org", "other-repo", nil)
	sig := sign(body, "doesnotmatter")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_MissingSignature_Returns401(t *testing.T) {
	s := NewServer(testConfig())
	body := buildPayload("acme", "myrepo", nil)
	w := postWebhook(t, s, "pull_request_review_comment", body, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_InvalidSignature_Returns401(t *testing.T) {
	s := NewServer(testConfig())
	body := buildPayload("acme", "myrepo", nil)
	w := postWebhook(t, s, "pull_request_review_comment", body, "sha256=deadbeef")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_WrongPrefixSignature_Returns401(t *testing.T) {
	s := NewServer(testConfig())
	body := buildPayload("acme", "myrepo", nil)
	w := postWebhook(t, s, "pull_request_review_comment", body, "sha1=abc123")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
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
