package workflow

import (
	"strings"

	"github.com/zon/ralph/internal/git"
	githubpkg "github.com/zon/ralph/internal/github"
)

// getRemoteURL gets the git remote URL.
func getRemoteURL() (string, error) {
	return git.GetRemoteURL()
}

// getRepoRoot gets the git repository root directory.
func getRepoRoot() (string, error) {
	return git.FindRepoRoot()
}

// getCurrentBranch gets the current git branch name.
func getCurrentBranch() (string, error) {
	return git.GetCurrentBranch()
}

// toHTTPSURL converts a GitHub SSH remote URL to HTTPS.
// SSH format: git@github.com:owner/repo.git -> https://github.com/owner/repo.git
// HTTPS URLs are returned unchanged.
func toHTTPSURL(remoteURL string) string {
	repo, err := githubpkg.ParseRemoteURL(remoteURL)
	if err != nil {
		// fallback to original logic
		if strings.HasPrefix(remoteURL, "git@github.com:") {
			return "https://github.com/" + strings.TrimPrefix(remoteURL, "git@github.com:")
		}
		return remoteURL
	}
	return repo.CloneURL()
}
