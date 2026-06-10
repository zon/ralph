package cmd

import (
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/orchestration/setup"
	"github.com/zon/ralph/internal/skills"
)

type SetCmd struct {
	Skills SetSkillsCmd `cmd:"" help:"Manage ralph skill installation"`
	Config SetConfigCmd `cmd:"" help:"Configure credentials for remote execution"`
}

type SetSkillsCmd struct {
	Branch string `help:"Ralph branch to install skills from" short:"b" name:"branch" optional:""`
}

type gitClient struct{}

func (gitClient) RepoRootOrCwd() string {
	return git.RepoRootOrCwd()
}

func (s *SetSkillsCmd) Run() error {
	branch := s.Branch
	if branch == "" {
		branch = "main"
	}

	svc := setup.New(&gitClient{}, setup.Skills{
		Discover:   skills.Discover,
		FetchAll:   skills.FetchAll,
		PruneStale: skills.PruneStale,
		InstallAll: skills.InstallAll,
	})

	return svc.SetSkills(branch)
}