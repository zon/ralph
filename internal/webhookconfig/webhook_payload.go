package webhookconfig

import (
	"strings"

	"github.com/zon/ralph/internal/github"
)

// IsAcceptable reports whether the payload should be dispatched for the given
// event type, applying user ignore/allowlist rules from cfg.
// Returns false for unrecognised event types, non-PR issue comments, empty
// review bodies, approved reviews, and non-commented review states. Approving
// a pull request is the reviewer's own action, so an approval requests no work.
func IsAcceptable(p *github.WebhookPayload, eventType string, cfg *Config) bool {
	repo := cfg.RepoByFullName(p.RepoOwner(), p.RepoName())

	switch eventType {
	case "issue_comment":
		if p.Issue.PullRequest == nil {
			return false
		}
		author := p.Comment.User.Login
		return !cfg.IsUserIgnored(repo, author) && (repo == nil || repo.IsUserAllowed(author))

	case "pull_request_review_comment":
		author := p.Comment.User.Login
		return !cfg.IsUserIgnored(repo, author) && (repo == nil || repo.IsUserAllowed(author))

	case "pull_request_review":
		author := p.Review.User.Login
		if cfg.IsUserIgnored(repo, author) || (repo != nil && !repo.IsUserAllowed(author)) {
			return false
		}
		switch strings.ToLower(p.Review.State) {
		case "approved":
			return false
		case "commented":
			return p.Review.Body != ""
		}
		return false
	}

	return false
}
