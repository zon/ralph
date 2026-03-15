package github

import "fmt"

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
