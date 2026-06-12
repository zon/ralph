package github

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zon/ralph/internal/webhookconfig"
)

// WebhookPayload is the subset of a GitHub webhook JSON payload that the server needs.
type WebhookPayload struct {
	Action string `json:"action"`
	Issue  struct {
		PullRequest *struct {
			URL string `json:"url"`
		} `json:"pull_request"`
	} `json:"issue"`
	PullRequest struct {
		Number int `json:"number"`
		Head   struct {
			Ref string `json:"ref"`
		} `json:"head"`
	} `json:"pull_request"`
	Comment struct {
		Body string `json:"body"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"comment"`
	Review struct {
		State string `json:"state"`
		Body  string `json:"body"`
		User  struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"review"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

// RepoOwner returns the repository owner login.
func (p *WebhookPayload) RepoOwner() string {
	return p.Repository.Owner.Login
}

// RepoName returns the repository name.
func (p *WebhookPayload) RepoName() string {
	return p.Repository.Name
}

// IsAcceptable reports whether the payload should be dispatched for the given
// event type, applying user ignore/allowlist rules from cfg.
// Returns false for unrecognised event types, non-PR issue comments, empty
// review bodies, and non-approved/non-commented review states.
func (p *WebhookPayload) IsAcceptable(eventType string, cfg *webhookconfig.Config) bool {
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
			return true
		case "commented":
			return p.Review.Body != ""
		}
		return false
	}

	return false
}

// EventFields holds the parsed fields from a WebhookPayload for a specific event type.
// The caller passes EventFields directly to workflow.FromWebhookEventWithConfig.
type EventFields struct {
	Body      string
	Approved  bool
	PRBranch  string
	RepoOwner string
	RepoName  string
	PRNumber  string
	Author    string
}

// ToEvent converts the payload into EventFields for the given event type.
// Call IsAcceptable first to ensure the payload is valid.
func (p *WebhookPayload) ToEvent(eventType string) EventFields {
	prNumber := ""
	if p.PullRequest.Number != 0 {
		prNumber = fmt.Sprintf("%d", p.PullRequest.Number)
	}
	branch := p.PullRequest.Head.Ref
	owner := p.RepoOwner()
	repoName := p.RepoName()

	switch eventType {
	case "issue_comment", "pull_request_review_comment":
		return EventFields{
			Body:      p.Comment.Body,
			PRBranch:  branch,
			RepoOwner: owner,
			RepoName:  repoName,
			PRNumber:  prNumber,
			Author:    p.Comment.User.Login,
		}
	case "pull_request_review":
		return EventFields{
			Body:      p.Review.Body,
			Approved:  strings.ToLower(p.Review.State) == "approved",
			PRBranch:  branch,
			RepoOwner: owner,
			RepoName:  repoName,
			PRNumber:  prNumber,
			Author:    p.Review.User.Login,
		}
	}
	return EventFields{}
}

// EventAction returns the action field from the payload.
func (p *WebhookPayload) EventAction() string {
	return p.Action
}

// PRNumber returns the pull request number.
func (p *WebhookPayload) PRNumber() int {
	return p.PullRequest.Number
}

// PRHeadRef returns the head branch ref of the pull request.
func (p *WebhookPayload) PRHeadRef() string {
	return p.PullRequest.Head.Ref
}

// CommentBody returns the comment body text.
func (p *WebhookPayload) CommentBody() string {
	return p.Comment.Body
}

// ReviewBody returns the review body text.
func (p *WebhookPayload) ReviewBody() string {
	return p.Review.Body
}

// ParseWebhookPayload unmarshals a raw JSON body into a WebhookPayload.
func ParseWebhookPayload(body []byte) (*WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON payload")
	}
	return &payload, nil
}


