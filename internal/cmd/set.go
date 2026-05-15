package cmd

import (
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/setup"
	"github.com/zon/ralph/internal/skills"
)

type SetCmd struct {
	Skills SetSkillsCmd `cmd:"" help:"Manage ralph skill installation"`
}

type SetSkillsCmd struct {
	Branch string `help:"Ralph branch to install skills from" short:"b" name:"branch" optional:""`
}

type gitClient struct{}

func (gitClient) RepoRoot() (string, error) {
	return git.RepoRoot()
}

type skillsClient struct{}

func (skillsClient) Discover(branch string) ([]string, error) {
	return skills.Discover(branch)
}

func (skillsClient) FetchAll(branch string, names []string) ([]skills.Skill, error) {
	return skills.FetchAll(branch, names)
}

func (skillsClient) PruneStale(root string, fetched []skills.Skill) {
	skills.PruneStale(root, fetched)
}

func (skillsClient) InstallAll(root string, fetched []skills.Skill) error {
	return skills.InstallAll(root, fetched)
}

func (s *SetSkillsCmd) Run() error {
	branch := s.Branch
	if branch == "" {
		branch = "main"
	}

	svc := setup.New(&gitClient{}, &skillsClient{})

	return svc.SetSkills(branch)
}