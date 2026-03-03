package cmd

import (
	"context"

	"github.com/zon/ralph/internal/github"
)

// Run executes the set-github-token command (implements kong.Run interface)
func (g *GithubTokenCmd) Run() error {
	return github.ConfigureGitAuth(context.Background(), g.Owner, g.Repo, g.SecretsDir)
}
