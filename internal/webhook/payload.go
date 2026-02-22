package webhook

import (
	"fmt"
	"strings"
)

// GithubPayload is the subset of a GitHub webhook JSON payload that the server needs.
type GithubPayload struct {
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
func (p GithubPayload) RepoOwner() string {
	return p.Repository.Owner.Login
}

// RepoName returns the repository name.
func (p GithubPayload) RepoName() string {
	return p.Repository.Name
}

// IsAcceptable reports whether the payload should be dispatched for the given
// event type, applying user ignore/allowlist rules from cfg.
// Returns false for unrecognised event types, non-PR issue comments, empty
// review bodies, and non-approved/non-commented review states.
func (p GithubPayload) IsAcceptable(eventType string, cfg *Config) bool {
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

// ToEvent converts the payload into an Event for the given event type.
// Call IsAcceptable first to ensure the payload is valid.
func (p GithubPayload) ToEvent(eventType string) Event {
	prNumber := ""
	if p.PullRequest.Number != 0 {
		prNumber = fmt.Sprintf("%d", p.PullRequest.Number)
	}
	branch := p.PullRequest.Head.Ref
	owner := p.RepoOwner()
	repoName := p.RepoName()

	switch eventType {
	case "issue_comment", "pull_request_review_comment":
		return Event{
			Body:      p.Comment.Body,
			PRBranch:  branch,
			RepoOwner: owner,
			RepoName:  repoName,
			PRNumber:  prNumber,
			Author:    p.Comment.User.Login,
		}
	case "pull_request_review":
		return Event{
			Body:      p.Review.Body,
			Approved:  strings.ToLower(p.Review.State) == "approved",
			PRBranch:  branch,
			RepoOwner: owner,
			RepoName:  repoName,
			PRNumber:  prNumber,
			Author:    p.Review.User.Login,
		}
	}
	return Event{}
}
