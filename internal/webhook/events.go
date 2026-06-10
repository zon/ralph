package webhook

// Event represents a filtered GitHub webhook event — either a comment or a review.
// Use IsComment and IsReview to distinguish them.
type Event struct {
	Body      string // Comment or review body text
	Approved  bool   // True only for approved pull_request_review events
	PRBranch  string // Head branch of the pull request
	RepoOwner string
	RepoName  string
	PRNumber  string
	Author    string // GitHub login of the commenter or reviewer
}

// IsComment reports whether the event is a comment (not an approval).
func (e Event) IsComment() bool {
	return !e.Approved
}

// IsReview reports whether the event is an approved pull request review.
func (e Event) IsReview() bool {
	return e.Approved
}
