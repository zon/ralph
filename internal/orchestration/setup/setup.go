package setup

import "github.com/zon/ralph/internal/skills"

type GitClient interface {
	RepoRootOrCwd() string
}

type Skills struct {
	Discover   func(branch string) ([]string, error)
	FetchAll   func(branch string, names []string) ([]skills.Skill, error)
	PruneStale func(root string, fetched []skills.Skill)
	InstallAll func(root string, fetched []skills.Skill) error
}

type Setup struct {
	skills Skills
	git    GitClient
}

func (s *Setup) SetSkills(branch string) error {
	root := s.git.RepoRootOrCwd()

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

func New(git GitClient, skills Skills) *Setup {
	return &Setup{git: git, skills: skills}
}
