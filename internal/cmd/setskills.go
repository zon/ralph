package cmd

import (
	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/skills"
)

type SetSkillsCmd struct {
	Branch string `help:"Branch to install skills from (default: main)" default:"main"`
}

func (c *SetSkillsCmd) Run() error {
	repoRoot, err := git.FindRepoRoot()
	if err != nil {
		return skills.ErrNotInGitRepo
	}

	discovered, err := skills.Discover(c.Branch)
	if err != nil {
		return err
	}

	contents, err := skills.FetchAll(discovered, c.Branch)
	if err != nil {
		return err
	}

	if err := skills.Install(repoRoot, contents); err != nil {
		return err
	}

	if err := skills.RemoveStale(repoRoot, discovered); err != nil {
		return err
	}

	logger.Infof("Installed %d skill(s) from branch '%s'", len(contents), c.Branch)
	return nil
}