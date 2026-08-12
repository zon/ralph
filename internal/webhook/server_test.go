package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/output"
	"github.com/zon/ralph/internal/webhookconfig"
	"gopkg.in/yaml.v3"
)

// testConfig builds a minimal Config suitable for server tests.
func testConfig() *webhookconfig.Config {
	return &webhookconfig.Config{
		App: webhookconfig.AppConfig{
			Port: 8080,
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "myrepo"},
			},
		},
		Secrets: webhookconfig.Secrets{
			Repos: []webhookconfig.RepoSecret{
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
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte("not json")))
	req.Header.Set("X-Hub-Signature-256", "sha256=anything")
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebhook_UnknownRepo_Returns401(t *testing.T) {
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	body := buildPayload("unknown-org", "other-repo", nil)
	sig := sign(body, "doesnotmatter")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_MissingSignature_Returns401(t *testing.T) {
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	body := buildPayload("acme", "myrepo", nil)
	w := postWebhook(t, s, "pull_request_review_comment", body, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_InvalidSignature_Returns401(t *testing.T) {
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	body := buildPayload("acme", "myrepo", nil)
	w := postWebhook(t, s, "pull_request_review_comment", body, "sha256=deadbeef")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_WrongPrefixSignature_Returns401(t *testing.T) {
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	body := buildPayload("acme", "myrepo", nil)
	w := postWebhook(t, s, "pull_request_review_comment", body, "sha1=abc123")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebhook_FilteredEvent_Returns200(t *testing.T) {
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	body := buildPayload("acme", "myrepo", nil)
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "unknown_event_type", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleWebhook_EmptyRepoOwner_Returns400(t *testing.T) {
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	body := buildPayload("", "myrepo", nil)
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebhook_EmptyRepoName_Returns400(t *testing.T) {
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	body := buildPayload("acme", "", nil)
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "pull_request_review_comment", body, sig)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebhook_ToWorkflowError_Returns200(t *testing.T) {
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	payload := map[string]interface{}{
		"repository": map[string]interface{}{
			"name": "myrepo",
			"owner": map[string]interface{}{
				"login": "acme",
			},
		},
		"comment": map[string]interface{}{
			"body": "hello",
			"user": map[string]interface{}{"login": "testuser"},
		},
		"issue": map[string]interface{}{
			"pull_request": map[string]interface{}{},
		},
	}
	body, _ := json.Marshal(payload)
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "issue_comment", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleWebhook_IssueComment_SubmitsWorkflow(t *testing.T) {
	submitCh := make(chan string, 1)
	mock := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submitCh <- workflowYAML
			return "test-workflow", nil
		},
	}
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), mock)

	payload := map[string]interface{}{
		"repository": map[string]interface{}{
			"name": "myrepo",
			"owner": map[string]interface{}{
				"login": "acme",
			},
		},
		"comment": map[string]interface{}{
			"body": "hello",
			"user": map[string]interface{}{"login": "testuser"},
		},
		"issue": map[string]interface{}{
			"pull_request": map[string]interface{}{},
		},
		"pull_request": map[string]interface{}{
			"number": 42,
			"head": map[string]interface{}{
				"ref": "ralph/my-feature",
			},
		},
	}
	body, _ := json.Marshal(payload)
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "issue_comment", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case workflowYAML := <-submitCh:
		assert.NotEmpty(t, workflowYAML)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for workflow submission")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Pull request review event tests
// ──────────────────────────────────────────────────────────────────────────────

// buildReviewPayload creates a pull_request_review payload for acme/myrepo with
// the given review state and body.
func buildReviewPayload(state, body string) []byte {
	payload := map[string]interface{}{
		"repository": map[string]interface{}{
			"name": "myrepo",
			"owner": map[string]interface{}{
				"login": "acme",
			},
		},
		"review": map[string]interface{}{
			"state": state,
			"body":  body,
			"user":  map[string]interface{}{"login": "testuser"},
		},
		"pull_request": map[string]interface{}{
			"number": 42,
			"head": map[string]interface{}{
				"ref": "ralph/my-feature",
			},
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

// postReview sends a pull_request_review event with the given state and body and
// returns the recorder.
func postReview(t *testing.T, s *Server, state, body string) *httptest.ResponseRecorder {
	t.Helper()
	payload := buildReviewPayload(state, body)
	sig := sign(payload, "supersecret")
	return postWebhook(t, s, "pull_request_review", payload, sig)
}

// noWorkflowSubmitted asserts that the mock was never asked to submit a workflow.
func noWorkflowSubmitted(t *testing.T, mock *argo.MockClient) {
	t.Helper()
	select {
	case <-time.After(200 * time.Millisecond):
	default:
	}
	assert.False(t, mock.SubmitYAMLCalled)
}

func TestHandleWebhook_ReviewApproved_Ignored(t *testing.T) {
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), &argo.MockClient{})
	w := postReview(t, s, "approved", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, w.Body.Len() > 0)
}

func TestHandleWebhook_ReviewApprovedWithBody_Ignored(t *testing.T) {
	mock := &argo.MockClient{}
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), mock)
	w := postReview(t, s, "approved", "This looks good to me")
	assert.Equal(t, http.StatusOK, w.Code)
	noWorkflowSubmitted(t, mock)
}

func TestHandleWebhook_ReviewChangesRequested_Ignored(t *testing.T) {
	mock := &argo.MockClient{}
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), mock)
	w := postReview(t, s, "changes_requested", "Please fix the tests")
	assert.Equal(t, http.StatusOK, w.Code)
	noWorkflowSubmitted(t, mock)
}

func TestHandleWebhook_ReviewCommented_SubmitsRunWorkflow(t *testing.T) {
	submitCh := make(chan string, 1)
	mock := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submitCh <- workflowYAML
			return "test-workflow", nil
		},
	}
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), mock)

	w := postReview(t, s, "commented", "Please add a test for the new helper")
	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case workflowYAML := <-submitCh:
		assert.Contains(t, workflowYAML, "ralph")
		assert.Contains(t, workflowYAML, "comment")
		assert.Contains(t, workflowYAML, "--comment-body")
		assert.Contains(t, workflowYAML, "Please add a test for the new helper")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for workflow submission")
	}
}

// Scenario: Run Workflow labeled.
//
// GIVEN a comment event triggers a Run Workflow submission
// WHEN the workflow YAML is rendered
// THEN the workflow metadata contains the label app.kubernetes.io/managed-by=ralph
func TestHandleWebhook_CommentEvent_SubmitsLabeledRunWorkflow(t *testing.T) {
	submitCh := make(chan string, 1)
	mock := &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submitCh <- workflowYAML
			return "test-workflow", nil
		},
	}
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), mock)

	payload := map[string]interface{}{
		"repository": map[string]interface{}{
			"name": "myrepo",
			"owner": map[string]interface{}{
				"login": "acme",
			},
		},
		"comment": map[string]interface{}{
			"body": "hello",
			"user": map[string]interface{}{"login": "testuser"},
		},
		"issue": map[string]interface{}{
			"pull_request": map[string]interface{}{},
		},
		"pull_request": map[string]interface{}{
			"number": 42,
			"head": map[string]interface{}{
				"ref": "ralph/my-feature",
			},
		},
	}
	body, _ := json.Marshal(payload)
	sig := sign(body, "supersecret")
	w := postWebhook(t, s, "issue_comment", body, sig)
	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case workflowYAML := <-submitCh:
		var wfData map[string]interface{}
		require.NoError(t, yaml.Unmarshal([]byte(workflowYAML), &wfData))

		metadata, ok := wfData["metadata"].(map[string]interface{})
		require.True(t, ok, "metadata is not a map")
		labels, ok := metadata["labels"].(map[string]interface{})
		require.True(t, ok, "metadata labels is not a map")
		assert.Equal(t, "ralph", labels["app.kubernetes.io/managed-by"], "workflow metadata should contain app.kubernetes.io/managed-by=ralph")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for workflow submission")
	}
}

func TestHandleWebhook_ReviewCommentedEmptyBody_Ignored(t *testing.T) {
	mock := &argo.MockClient{}
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), mock)
	w := postReview(t, s, "commented", "")
	assert.Equal(t, http.StatusOK, w.Code)
	noWorkflowSubmitted(t, mock)
}

// ──────────────────────────────────────────────────────────────────────────────
// Item tests: no merge workflow submission for any event
// ──────────────────────────────────────────────────────────────────────────────

// TestHandleWebhook_NoEventSubmitsMergeWorkflow asserts that the server has no
// branch that submits a merge workflow: an approved review, which used to trigger
// a merge, submits no workflow at all, and a commented review submits a run
// workflow whose rendered YAML never references a merge.
func TestHandleWebhook_NoEventSubmitsMergeWorkflow(t *testing.T) {
	mock := &argo.MockClient{}
	s := NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), mock)

	w := postReview(t, s, "approved", "")
	assert.Equal(t, http.StatusOK, w.Code)
	noWorkflowSubmitted(t, mock)

	submitCh := make(chan string, 1)
	mock = &argo.MockClient{
		SubmitYAMLFunc: func(ctx context.Context, workflowYAML string, kubeCtx argo.K8sContext) (string, error) {
			submitCh <- workflowYAML
			return "test-workflow", nil
		},
	}
	s = NewServer(testConfig(), output.NewClient(os.Stdout, os.Stderr, false), mock)

	_ = postReview(t, s, "commented", "Please add a test")
	select {
	case workflowYAML := <-submitCh:
		assert.NotContains(t, workflowYAML, "merge")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for workflow submission")
	}
}

// TestHandleWebhook_ApprovedReview_NoWorkflowSubmissionLogLine asserts that an
// approved review produces no log line claiming a workflow was submitted.
func TestHandleWebhook_ApprovedReview_NoWorkflowSubmissionLogLine(t *testing.T) {
	var logBuf bytes.Buffer
	mock := &argo.MockClient{}
	s := NewServer(testConfig(), output.NewClient(&logBuf, os.Stderr, true), mock)

	w := postReview(t, s, "approved", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, mock.SubmitYAMLCalled)
	assert.NotContains(t, logBuf.String(), "submitted")
}
