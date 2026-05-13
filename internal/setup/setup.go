package setup

import (
	"fmt"

	"github.com/zon/ralph/internal/git"
	"github.com/zon/ralph/internal/skills"
)

func SetSkills(branch string) error {
	root, err := git.RepoRoot()
	if err != nil {
		return err
	}

	names, err := skills.Discover(branch)
	if err != nil {
		return err
	}

	fetched, err := skills.FetchAll(branch, names)
	if err != nil {
		return err
	}

	skills.PruneStale(root, fetched)
	return skills.InstallAll(root, fetched)
}

func SetSkillsWithClient(branch string, httpClient interface{}) error {
	return fmt.Errorf("not implemented")
}