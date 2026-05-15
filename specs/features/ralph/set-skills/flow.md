# Set Skills Flow

## Purpose
Discover, fetch, and install ralph-prefixed Claude Code skills from the ralph GitHub repository into the target git repository, pruning any stale ralph skills no longer present upstream.

## Flow

**Module:** `internal/setup`

```go
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
```

### Helpers

- **`s.git.RepoRoot()`** — returns the root path of the git repository containing the current working directory
- **`s.skills.Discover(branch)`** — queries the GitHub Contents API and returns the names of all ralph-prefixed skill directories on the given branch
- **`s.skills.FetchAll(branch, names)`** — fetches each skill's SKILL.md from raw GitHub URLs, rewrites links to the resolved branch, and returns the collected skills
- **`s.skills.PruneStale(root, fetched)`** — removes ralph-prefixed skill directories from the target repository that are absent from the fetched set; leaves non-ralph skills untouched
- **`s.skills.InstallAll(root, fetched)`** — writes each fetched skill's SKILL.md into `.claude/skills/<name>/` under the target root, creating the directory if absent

## Tests

**Module:** `internal/setup`

```go
func TestSetSkillsSuccess(t *testing.T) {
	root := git.anyRoot()
	fetched := skills.anySkills()
	mock := skills.newMock(skills.thatFetches(fetched))
	svc := setup.withMocks(
		setup.withGit(git.thatFindsRoot(root)),
		setup.withSkills(mock),
	)
	require.NoError(t, svc.SetSkills("main"))
	require.Equal(t, fetched, mock.installed())
	require.Equal(t, fetched, mock.pruned())
}

func TestSetSkillsNoGitRepo(t *testing.T) {
	mock := skills.newMock()
	svc := setup.withMocks(
		setup.withGit(git.withNoRepo()),
		setup.withSkills(mock),
	)
	require.Error(t, svc.SetSkills("main"))
	require.Empty(t, mock.installed())
}

func TestSetSkillsDiscoveryFails(t *testing.T) {
	mock := skills.newMock(skills.thatFailsDiscovery())
	svc := setup.withMocks(setup.withSkills(mock))
	require.Error(t, svc.SetSkills("main"))
	require.Empty(t, mock.installed())
}

func TestSetSkillsFetchFails(t *testing.T) {
	mock := skills.newMock(skills.thatFailsFetch())
	svc := setup.withMocks(setup.withSkills(mock))
	require.Error(t, svc.SetSkills("main"))
	require.Empty(t, mock.installed())
}
```

### Helpers

- **`setup.withMocks(opts...)`** — constructs a `Setup` with default mock implementations; pass option helpers to configure specific dependencies
- **`setup.withGit(client)`** — option that sets the git client on the mock setup
- **`setup.withSkills(client)`** — option that sets the skills client on the mock setup
- **`git.anyRoot()`** — returns a valid repository root path suitable for use in tests
- **`git.thatFindsRoot(root)`** — returns a git mock whose `RepoRoot` returns the given root
- **`git.withNoRepo()`** — returns a git mock whose `RepoRoot` returns an error
- **`skills.anySkills()`** — returns a non-empty slice of skills in a default valid state
- **`skills.newMock(opts...)`** — constructs a recording skills mock; pass option helpers to configure its behavior
- **`skills.thatFetches(skills)`** — option configuring the mock to return names derived from the given skills on `Discover` and the skills themselves on `FetchAll`
- **`skills.thatFailsDiscovery()`** — option configuring the mock to return an error from `Discover`
- **`skills.thatFailsFetch()`** — option configuring the mock to return an error from `FetchAll`
- **`mock.installed()`** — returns the skills passed to `InstallAll` during the test
- **`mock.pruned()`** — returns the skills passed to `PruneStale` during the test
