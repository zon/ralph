package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zon/ralph/internal/webhookconfig"
)

// minimalPayload builds a GithubPayload for acme/myrepo with the given overrides.
func minimalPayload(owner, repo string) GithubPayload {
	var p GithubPayload
	p.Repository.Owner.Login = owner
	p.Repository.Name = repo
	return p
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
	// Issue.PullRequest is nil → plain issue comment
	assert.False(t, p.IsAcceptable("issue_comment", openConfig()))
}

func TestIsAcceptable_IssueComment_OnPR_ReturnsTrue(t *testing.T) {
	p := minimalPayload("acme", "myrepo")
	p.Comment.User.Login = "alice"
	p.Issue.PullRequest = &struct{ URL string `json:"url"` }{URL: "https://example.com"}
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
	assert.True(t, e.IsComment())
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
	assert.True(t, e.IsReview())
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
	// PullRequest.Number is zero
	e := p.ToEvent("pull_request_review_comment")
	assert.Equal(t, "", e.PRNumber)
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

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
