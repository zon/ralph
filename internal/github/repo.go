package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/zon/ralph/internal/git"
)

// Repo represents a GitHub repository.
type Repo struct {
	Owner string
	Name  string
}

// MakeRepo creates a Repo with the given owner and name.
func MakeRepo(owner, name string) Repo {
	return Repo{Owner: owner, Name: name}
}

// CloneURL returns the HTTPS GitHub clone URL for the repository.
func (r Repo) CloneURL() string {
	return fmt.Sprintf("https://github.com/%s/%s.git", r.Owner, r.Name)
}

// CloneURL returns the HTTPS GitHub clone URL for a repository.
func CloneURL(owner, name string) string {
	return Repo{Owner: owner, Name: name}.CloneURL()
}

// GetRepo extracts the repository owner and name from git remote origin.
func GetRepo(ctx context.Context) (Repo, error) {
	remoteURL, err := git.RemoteURL()
	if err != nil {
		return Repo{}, fmt.Errorf("failed to get remote.origin.url: %w", err)
	}

	return ParseRemoteURL(remoteURL)
}

// ParseRemoteURL parses a GitHub remote URL and returns the repository.
// Supported formats:
//
//	git@github.com:owner/repo.git
//	https://github.com/owner/repo.git
//	https://github.com/owner/repo
func ParseRemoteURL(remoteURL string) (Repo, error) {
	if remoteURL == "" {
		return Repo{}, fmt.Errorf("remote.origin.url is empty")
	}

	var repoPath string

	if strings.HasPrefix(remoteURL, "git@github.com:") {
		repoPath = strings.TrimPrefix(remoteURL, "git@github.com:")
	} else if strings.Contains(remoteURL, "github.com/") {
		parts := strings.Split(remoteURL, "github.com/")
		if len(parts) > 1 {
			repoPath = parts[1]
		}
	} else {
		return Repo{}, fmt.Errorf("not a GitHub repository URL: %s", remoteURL)
	}

	repoPath = strings.TrimSuffix(repoPath, ".git")

	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return Repo{}, fmt.Errorf("invalid repository path: %s", repoPath)
	}

	return MakeRepo(parts[0], parts[1]), nil
}
