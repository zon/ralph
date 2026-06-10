package setup

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/skills"
)

var ErrDiscoveryFailed = errors.New("discovery failed")
var ErrFetchFailed = errors.New("fetch failed")

type mockGit struct {
	rootFn func() string
}

func (m *mockGit) RepoRootOrCwd() string {
	return m.rootFn()
}

type gitClientFunc func() string

func (f gitClientFunc) RepoRootOrCwd() string {
	return f()
}

func thatFindsRoot(root string) GitClient {
	return gitClientFunc(func() string { return root })
}

func withNoRepo() GitClient {
	cwd, _ := os.Getwd()
	return gitClientFunc(func() string { return cwd })
}

type deps struct {
	git    GitClient
	skills Skills
}

type Opt func(*deps)

func withGit(gitClient GitClient) Opt {
	return func(d *deps) {
		d.git = gitClient
	}
}

func withSkills(skillsClient Skills) Opt {
	return func(d *deps) {
		d.skills = skillsClient
	}
}

func withMocks(opts ...Opt) *Setup {
	d := &deps{}
	for _, opt := range opts {
		opt(d)
	}
	return &Setup{
		git:    d.git,
		skills: d.skills,
	}
}

func anyRoot() string {
	return "/fake/root"
}

func anySkills() []skills.Skill {
	return []skills.Skill{
		{Name: "ralph-test-skill", Content: "# Test Skill"},
	}
}

func TestSetSkillsSuccess(t *testing.T) {
	root := anyRoot()
	fetched := anySkills()
	var installed, pruned []skills.Skill
	mock := Skills{
		Discover: func(branch string) ([]string, error) { return nil, nil },
		FetchAll: func(branch string, names []string) ([]skills.Skill, error) {
			return fetched, nil
		},
		PruneStale: func(root string, f []skills.Skill) {
			pruned = f
		},
		InstallAll: func(root string, f []skills.Skill) error {
			installed = f
			return nil
		},
	}
	svc := withMocks(
		withGit(thatFindsRoot(root)),
		withSkills(mock),
	)
	require.NoError(t, svc.SetSkills("main"))
	require.Equal(t, fetched, installed)
	require.Equal(t, fetched, pruned)
}

func TestSetSkillsNoGitRepo(t *testing.T) {
	fetched := anySkills()
	var installed []skills.Skill
	mock := Skills{
		Discover: func(branch string) ([]string, error) { return nil, nil },
		FetchAll: func(branch string, names []string) ([]skills.Skill, error) {
			return fetched, nil
		},
		PruneStale: func(root string, f []skills.Skill) {},
		InstallAll: func(root string, f []skills.Skill) error {
			installed = f
			return nil
		},
	}
	svc := withMocks(
		withGit(withNoRepo()),
		withSkills(mock),
	)
	require.NoError(t, svc.SetSkills("main"))
	require.Equal(t, fetched, installed)
}

func TestSetSkillsDiscoveryFails(t *testing.T) {
	var installed []skills.Skill
	mock := Skills{
		Discover: func(branch string) ([]string, error) {
			return nil, ErrDiscoveryFailed
		},
	}
	svc := withMocks(
		withGit(thatFindsRoot(anyRoot())),
		withSkills(mock),
	)
	require.Error(t, svc.SetSkills("main"))
	require.Empty(t, installed)
}

func TestSetSkillsFetchFails(t *testing.T) {
	var installed []skills.Skill
	mock := Skills{
		Discover: func(branch string) ([]string, error) { return nil, nil },
		FetchAll: func(branch string, names []string) ([]skills.Skill, error) {
			return nil, ErrFetchFailed
		},
	}
	svc := withMocks(
		withGit(thatFindsRoot(anyRoot())),
		withSkills(mock),
	)
	require.Error(t, svc.SetSkills("main"))
	require.Empty(t, installed)
}
