package setup

import "github.com/zon/ralph/internal/skills"

type GitClient interface {
	RepoRoot() (string, error)
}

type SkillsClient interface {
	Discover(branch string) ([]string, error)
	FetchAll(branch string, names []string) ([]skills.Skill, error)
	PruneStale(root string, fetched []skills.Skill)
	InstallAll(root string, fetched []skills.Skill) error
}

type Setup struct {
	skills SkillsClient
	git    GitClient
}

func (s *Setup) SetSkills(branch string) error {
	root, err := s.git.RepoRoot()
	if err != nil {
		return err
	}
	names, err := s.skills.Discover(branch)
	if err != nil {
		return err
	}
	fetched, err := s.skills.FetchAll(branch, names)
	if err != nil {
		return err
	}
	s.skills.PruneStale(root, fetched)
	return s.skills.InstallAll(root, fetched)
}