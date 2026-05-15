package setup

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/skills"
)

var ErrDiscoveryFailed = errors.New("discovery failed")
var ErrFetchFailed = errors.New("fetch failed")

type mockGit struct {
	rootFn func() (string, error)
}

func (m *mockGit) RepoRoot() (string, error) {
	return m.rootFn()
}

type gitClientFunc func() (string, error)

func (f gitClientFunc) RepoRoot() (string, error) {
	return f()
}

func thatFindsRoot(root string) GitClient {
	return gitClientFunc(func() (string, error) { return root, nil })
}

func withNoRepo() GitClient {
	return gitClientFunc(func() (string, error) { return "", errors.New("not inside a git repository") })
}

type mockSkills struct {
	discoverFn   func(branch string) ([]string, error)
	fetchAllFn   func(branch string, names []string) ([]skills.Skill, error)
	pruneStaleFn func(root string, fetched []skills.Skill)
	installAllFn func(root string, fetched []skills.Skill) error
	installedVar []skills.Skill
	prunedVar    []skills.Skill
}

func newMock(opts ...func(*mockSkills)) *mockSkills {
	m := &mockSkills{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *mockSkills) Discover(branch string) ([]string, error) {
	if m.discoverFn != nil {
		return m.discoverFn(branch)
	}
	return nil, nil
}

func (m *mockSkills) FetchAll(branch string, names []string) ([]skills.Skill, error) {
	if m.fetchAllFn != nil {
		return m.fetchAllFn(branch, names)
	}
	return nil, nil
}

func (m *mockSkills) PruneStale(root string, fetched []skills.Skill) {
	m.prunedVar = fetched
	if m.pruneStaleFn != nil {
		m.pruneStaleFn(root, fetched)
	}
}

func (m *mockSkills) InstallAll(root string, fetched []skills.Skill) error {
	m.installedVar = fetched
	if m.installAllFn != nil {
		return m.installAllFn(root, fetched)
	}
	return nil
}

func (m *mockSkills) installed() []skills.Skill {
	return m.installedVar
}

func (m *mockSkills) pruned() []skills.Skill {
	return m.prunedVar
}

func thatFetches(fetched []skills.Skill) func(*mockSkills) {
	return func(m *mockSkills) {
		m.fetchAllFn = func(branch string, names []string) ([]skills.Skill, error) {
			return fetched, nil
		}
	}
}

func thatFailsDiscovery() func(*mockSkills) {
	return func(m *mockSkills) {
		m.discoverFn = func(branch string) ([]string, error) {
			return nil, ErrDiscoveryFailed
		}
	}
}

func thatFailsFetch() func(*mockSkills) {
	return func(m *mockSkills) {
		m.fetchAllFn = func(branch string, names []string) ([]skills.Skill, error) {
			return nil, ErrFetchFailed
		}
	}
}

type deps struct {
	git    GitClient
	skills SkillsClient
}

type Opt func(*deps)

func withGit(gitClient GitClient) Opt {
	return func(d *deps) {
		d.git = gitClient
	}
}

func withSkills(skillsClient SkillsClient) Opt {
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
	mock := newMock(thatFetches(fetched))
	svc := withMocks(
		withGit(thatFindsRoot(root)),
		withSkills(mock),
	)
	require.NoError(t, svc.SetSkills("main"))
	require.Equal(t, fetched, mock.installed())
	require.Equal(t, fetched, mock.pruned())
}

func TestSetSkillsNoGitRepo(t *testing.T) {
	mock := newMock()
	svc := withMocks(
		withGit(withNoRepo()),
		withSkills(mock),
	)
	require.Error(t, svc.SetSkills("main"))
	require.Empty(t, mock.installed())
}

func TestSetSkillsDiscoveryFails(t *testing.T) {
	mock := newMock(thatFailsDiscovery())
	svc := withMocks(
		withGit(thatFindsRoot(anyRoot())),
		withSkills(mock),
	)
	require.Error(t, svc.SetSkills("main"))
	require.Empty(t, mock.installed())
}

func TestSetSkillsFetchFails(t *testing.T) {
	mock := newMock(thatFailsFetch())
	svc := withMocks(
		withGit(thatFindsRoot(anyRoot())),
		withSkills(mock),
	)
	require.Error(t, svc.SetSkills("main"))
	require.Empty(t, mock.installed())
}