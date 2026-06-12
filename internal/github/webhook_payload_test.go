package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/webhookconfig"
)

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// minimalPayload builds a WebhookPayload for acme/myrepo with the given overrides.
func minimalPayload(owner, repo string) WebhookPayload {
	var p WebhookPayload
	p.Repository.Owner.Login = owner
	p.Repository.Name = repo
	return p
}

// openConfig returns a Config with no allowlist restrictions.
func openConfig() *webhookconfig.Config {
	return &webhookconfig.Config{
		App: webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "myrepo"},
			},
		},
	}
}

// configWithAllowedUsers returns a Config with an AllowedUsers list on acme/myrepo.
func configWithAllowedUsers(users []string) *webhookconfig.Config {
	cfg := openConfig()
	cfg.App.Repos[0].AllowedUsers = users
	return cfg
}

// ──────────────────────────────────────────────────────────────────────────────
// IsAcceptable tests
// ──────────────────────────────────────────────────────────────────────────────

func TestIsAcceptable_UnknownEventType_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	assert.False(t, p.IsAcceptable("push", openConfig()))
}

func TestIsAcceptable_IssueComment_NonPR_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "alice"
	assert.False(t, p.IsAcceptable("issue_comment", openConfig()))
}

func TestIsAcceptable_IssueComment_OnPR_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "alice"
	p.Issue.PullRequest = &struct {
		URL string `json:"url"`
	}{URL: "https://example.com"}
	assert.True(t, p.IsAcceptable("issue_comment", openConfig()))
}

func TestIsAcceptable_PullRequestReviewComment_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "alice"
	assert.True(t, p.IsAcceptable("pull_request_review_comment", openConfig()))
}

func TestIsAcceptable_ReviewApproved_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "approved"
	p.Review.User.Login = "alice"
	assert.True(t, p.IsAcceptable("pull_request_review", openConfig()))
}

func TestIsAcceptable_ReviewCommentedWithBody_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "commented"
	p.Review.Body = "please fix this"
	p.Review.User.Login = "alice"
	assert.True(t, p.IsAcceptable("pull_request_review", openConfig()))
}

func TestIsAcceptable_ReviewCommentedEmptyBody_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "commented"
	p.Review.Body = ""
	p.Review.User.Login = "alice"
	assert.False(t, p.IsAcceptable("pull_request_review", openConfig()))
}

func TestIsAcceptable_ReviewChangesRequested_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "changes_requested"
	p.Review.User.Login = "alice"
	assert.False(t, p.IsAcceptable("pull_request_review", openConfig()))
}

func TestIsAcceptable_IgnoredUser_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "bot"
	cfg := openConfig()
	cfg.App.RalphUser = "bot"
	assert.False(t, p.IsAcceptable("pull_request_review_comment", cfg))
}

func TestIsAcceptable_UserNotInAllowedList_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "charlie"
	cfg := configWithAllowedUsers([]string{"alice", "bob"})
	assert.False(t, p.IsAcceptable("pull_request_review_comment", cfg))
}

func TestIsAcceptable_UserInAllowedList_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "alice"
	cfg := configWithAllowedUsers([]string{"alice", "bob"})
	assert.True(t, p.IsAcceptable("pull_request_review_comment", cfg))
}

func TestIsAcceptable_ReviewApprovedCaseInsensitive_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "APPROVED"
	p.Review.User.Login = "alice"
	assert.True(t, p.IsAcceptable("pull_request_review", openConfig()))
}

func TestIsAcceptable_IssueComment_IgnoredRalphUser_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "bot"
	p.Issue.PullRequest = &struct {
		URL string `json:"url"`
	}{URL: "https://example.com"}
	cfg := openConfig()
	cfg.App.RalphUser = "bot"
	assert.False(t, p.IsAcceptable("issue_comment", cfg))
}

func TestIsAcceptable_IssueComment_UserNotInAllowedList_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "charlie"
	p.Issue.PullRequest = &struct {
		URL string `json:"url"`
	}{URL: "https://example.com"}
	cfg := configWithAllowedUsers([]string{"alice", "bob"})
	assert.False(t, p.IsAcceptable("issue_comment", cfg))
}

func TestIsAcceptable_IssueComment_EmptyAllowedList_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "charlie"
	p.Issue.PullRequest = &struct {
		URL string `json:"url"`
	}{URL: "https://example.com"}
	cfg := configWithAllowedUsers([]string{})
	assert.True(t, p.IsAcceptable("issue_comment", cfg))
}

func TestIsAcceptable_ReviewApproved_IgnoredRalphUser_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "approved"
	p.Review.User.Login = "bot"
	cfg := openConfig()
	cfg.App.RalphUser = "bot"
	assert.False(t, p.IsAcceptable("pull_request_review", cfg))
}

func TestIsAcceptable_ReviewApproved_UserNotInAllowedList_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "approved"
	p.Review.User.Login = "charlie"
	cfg := configWithAllowedUsers([]string{"alice", "bob"})
	assert.False(t, p.IsAcceptable("pull_request_review", cfg))
}

func TestIsAcceptable_ReviewApproved_EmptyAllowedList_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "approved"
	p.Review.User.Login = "charlie"
	cfg := configWithAllowedUsers([]string{})
	assert.True(t, p.IsAcceptable("pull_request_review", cfg))
}

// ──────────────────────────────────────────────────────────────────────────────
// ToEvent tests
// ──────────────────────────────────────────────────────────────────────────────

func TestToEvent_IssueComment_FieldsPopulated(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.Body = "please fix this"
	p.Comment.User.Login = "alice"
	p.PullRequest.Number = 42
	p.PullRequest.Head.Ref = "ralph/my-feature"

	e := p.ToEvent("issue_comment")

	assert.Equal(t, "please fix this", e.Body)
	assert.Equal(t, "alice", e.Author)
	assert.Equal(t, "42", e.PRNumber)
	assert.Equal(t, "ralph/my-feature", e.PRBranch)
	assert.Equal(t, "acme", e.RepoOwner)
	assert.Equal(t, "myrepo", e.RepoName)
	assert.False(t, e.Approved)
}

func TestToEvent_PullRequestReviewComment_FieldsPopulated(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.Body = "nit: rename this"
	p.Comment.User.Login = "bob"
	p.PullRequest.Head.Ref = "ralph/my-feature"

	e := p.ToEvent("pull_request_review_comment")

	assert.Equal(t, "nit: rename this", e.Body)
	assert.Equal(t, "bob", e.Author)
	assert.False(t, e.Approved)
}

func TestToEvent_ReviewApproved_ApprovedTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "approved"
	p.Review.User.Login = "carol"
	p.PullRequest.Head.Ref = "ralph/my-feature"

	e := p.ToEvent("pull_request_review")

	assert.True(t, e.Approved)
	assert.Equal(t, "carol", e.Author)
}

func TestToEvent_ReviewCommented_ApprovedFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "commented"
	p.Review.Body = "looks good overall"
	p.Review.User.Login = "dave"

	e := p.ToEvent("pull_request_review")

	assert.False(t, e.Approved)
	assert.Equal(t, "looks good overall", e.Body)
}

func TestToEvent_NoPRNumber_EmptyString(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.Body = "hello"
	e := p.ToEvent("pull_request_review_comment")
	assert.Equal(t, "", e.PRNumber)
}

func TestToEvent_UnknownEventType_ReturnsEmptyEvent(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.Body = "test"
	p.Review.State = "approved"
	p.Review.User.Login = "alice"
	p.PullRequest.Number = 42

	e := p.ToEvent("push")

	assert.Equal(t, "", e.Body)
	assert.Equal(t, "", e.Author)
	assert.Equal(t, "", e.PRNumber)
	assert.Equal(t, "", e.PRBranch)
	assert.Equal(t, "", e.RepoOwner)
	assert.Equal(t, "", e.RepoName)
	assert.False(t, e.Approved)
}

func TestToEvent_ReviewChangesRequested_ApprovedFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "changes_requested"
	p.Review.User.Login = "carol"
	p.PullRequest.Head.Ref = "ralph/my-feature"

	e := p.ToEvent("pull_request_review")

	assert.False(t, e.Approved)
	assert.Equal(t, "carol", e.Author)
}

// ──────────────────────────────────────────────────────────────────────────────
// ParseWebhookPayload tests
// ──────────────────────────────────────────────────────────────────────────────

func TestParseWebhookPayload_ValidJSON_ParsesCorrectly(t *testing.T) {
	body := []byte(`{"repository":{"name":"myrepo","owner":{"login":"acme"}}}`)
	p, err := ParseWebhookPayload(body)
	require.NoError(t, err)
	assert.Equal(t, "acme", p.RepoOwner())
	assert.Equal(t, "myrepo", p.RepoName())
}

func TestParseWebhookPayload_InvalidJSON_ReturnsError(t *testing.T) {
	_, err := ParseWebhookPayload([]byte("not json"))
	assert.Error(t, err)
}

func TestParseWebhookPayload_EmptyBody_ReturnsError(t *testing.T) {
	_, err := ParseWebhookPayload([]byte{})
	assert.Error(t, err)
}
