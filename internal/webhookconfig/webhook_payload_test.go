package webhookconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zon/ralph/internal/github"
)

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// minimalPayload builds a WebhookPayload for acme/myrepo with the given overrides.
func minimalPayload(owner, repo string) github.WebhookPayload {
	var p github.WebhookPayload
	p.Repository.Owner.Login = owner
	p.Repository.Name = repo
	return p
}

// openConfig returns a Config with no allowlist restrictions.
func openConfig() *Config {
	return &Config{
		App: AppConfig{
			Repos: []RepoConfig{
				{Owner: "acme", Name: "myrepo"},
			},
		},
	}
}

// configWithAllowedUsers returns a Config with an AllowedUsers list on acme/myrepo.
func configWithAllowedUsers(users []string) *Config {
	cfg := openConfig()
	cfg.App.Repos[0].AllowedUsers = users
	return cfg
}

// ──────────────────────────────────────────────────────────────────────────────
// IsAcceptable tests
// ──────────────────────────────────────────────────────────────────────────────

func TestIsAcceptable_UnknownEventType_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	assert.False(t, IsAcceptable(&p, "push", openConfig()))
}

func TestIsAcceptable_IssueComment_NonPR_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "alice"
	assert.False(t, IsAcceptable(&p, "issue_comment", openConfig()))
}

func TestIsAcceptable_IssueComment_OnPR_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "alice"
	p.Issue.PullRequest = &struct {
		URL string `json:"url"`
	}{URL: "https://example.com"}
	assert.True(t, IsAcceptable(&p, "issue_comment", openConfig()))
}

func TestIsAcceptable_PullRequestReviewComment_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "alice"
	assert.True(t, IsAcceptable(&p, "pull_request_review_comment", openConfig()))
}

func TestIsAcceptable_ReviewApproved_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "approved"
	p.Review.User.Login = "alice"
	assert.True(t, IsAcceptable(&p, "pull_request_review", openConfig()))
}

func TestIsAcceptable_ReviewCommentedWithBody_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "commented"
	p.Review.Body = "please fix this"
	p.Review.User.Login = "alice"
	assert.True(t, IsAcceptable(&p, "pull_request_review", openConfig()))
}

func TestIsAcceptable_ReviewCommentedEmptyBody_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "commented"
	p.Review.Body = ""
	p.Review.User.Login = "alice"
	assert.False(t, IsAcceptable(&p, "pull_request_review", openConfig()))
}

func TestIsAcceptable_ReviewChangesRequested_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "changes_requested"
	p.Review.User.Login = "alice"
	assert.False(t, IsAcceptable(&p, "pull_request_review", openConfig()))
}

func TestIsAcceptable_IgnoredUser_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "bot"
	cfg := openConfig()
	cfg.App.RalphUser = "bot"
	assert.False(t, IsAcceptable(&p, "pull_request_review_comment", cfg))
}

func TestIsAcceptable_UserNotInAllowedList_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "charlie"
	cfg := configWithAllowedUsers([]string{"alice", "bob"})
	assert.False(t, IsAcceptable(&p, "pull_request_review_comment", cfg))
}

func TestIsAcceptable_UserInAllowedList_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "alice"
	cfg := configWithAllowedUsers([]string{"alice", "bob"})
	assert.True(t, IsAcceptable(&p, "pull_request_review_comment", cfg))
}

func TestIsAcceptable_ReviewApprovedCaseInsensitive_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "APPROVED"
	p.Review.User.Login = "alice"
	assert.True(t, IsAcceptable(&p, "pull_request_review", openConfig()))
}

func TestIsAcceptable_IssueComment_IgnoredRalphUser_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "bot"
	p.Issue.PullRequest = &struct {
		URL string `json:"url"`
	}{URL: "https://example.com"}
	cfg := openConfig()
	cfg.App.RalphUser = "bot"
	assert.False(t, IsAcceptable(&p, "issue_comment", cfg))
}

func TestIsAcceptable_IssueComment_UserNotInAllowedList_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "charlie"
	p.Issue.PullRequest = &struct {
		URL string `json:"url"`
	}{URL: "https://example.com"}
	cfg := configWithAllowedUsers([]string{"alice", "bob"})
	assert.False(t, IsAcceptable(&p, "issue_comment", cfg))
}

func TestIsAcceptable_IssueComment_EmptyAllowedList_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "charlie"
	p.Issue.PullRequest = &struct {
		URL string `json:"url"`
	}{URL: "https://example.com"}
	cfg := configWithAllowedUsers([]string{})
	assert.True(t, IsAcceptable(&p, "issue_comment", cfg))
}

func TestIsAcceptable_ReviewApproved_IgnoredRalphUser_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "approved"
	p.Review.User.Login = "bot"
	cfg := openConfig()
	cfg.App.RalphUser = "bot"
	assert.False(t, IsAcceptable(&p, "pull_request_review", cfg))
}

func TestIsAcceptable_ReviewApproved_UserNotInAllowedList_ReturnsFalse(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "approved"
	p.Review.User.Login = "charlie"
	cfg := configWithAllowedUsers([]string{"alice", "bob"})
	assert.False(t, IsAcceptable(&p, "pull_request_review", cfg))
}

func TestIsAcceptable_ReviewApproved_EmptyAllowedList_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Review.State = "approved"
	p.Review.User.Login = "charlie"
	cfg := configWithAllowedUsers([]string{})
	assert.True(t, IsAcceptable(&p, "pull_request_review", cfg))
}
